package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/usecase"
)

// Límites de análisis con IA. Cada uno consume cuota de Gemini, así que se
// limita por usuario (3 seguidos y luego 1 cada 20 min) y en todo el servidor
// (10 seguidos y luego 1 cada 6 s, ~10/min), por debajo del límite de la llave.
const (
	aiUserBurst      = 3
	aiUserInterval   = 20 * time.Minute
	aiGlobalBurst    = 10
	aiGlobalInterval = 6 * time.Second
	// aiGlobalKey es la clave del limitador global (un único bucket).
	aiGlobalKey = "global"
	// aiWriteTimeout amplía el WriteTimeout del servidor solo para el análisis,
	// que puede tardar varios reintentos de la IA.
	aiWriteTimeout = 60 * time.Second
)

// InsightService es el caso de uso de análisis con IA que expone la API.
type InsightService interface {
	Diagnose(ctx context.Context, userID domain.ID) (usecase.InsightReport, error)
}

type aiStatusResponse struct {
	// Enabled es false si el servidor no tiene llave de Gemini configurada.
	Enabled bool `json:"enabled"`
}

type diagnosisResponse struct {
	Headline        string               `json:"headline"`
	Summary         string               `json:"summary"`
	FinanceInsights []string             `json:"finance_insights"`
	HabitInsights   []string             `json:"habit_insights"`
	Suggestions     []suggestionResponse `json:"suggestions"`
	Period          periodResponse       `json:"period"`
	GeneratedAt     time.Time            `json:"generated_at"`
}

type suggestionResponse struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

type periodResponse struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// aiStatus atiende GET /api/v1/ai.
func (a *api) aiStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, aiStatusResponse{Enabled: a.Insights != nil})
}

// diagnose atiende POST /api/v1/ai/diagnose: genera un diagnóstico de las
// finanzas y los hábitos del usuario con IA.
func (a *api) diagnose(w http.ResponseWriter, r *http.Request) {
	if a.Insights == nil {
		writeError(w, http.StatusServiceUnavailable, "ai_disabled", "el análisis con IA no está configurado en el servidor")
		return
	}
	userID := userIDFrom(r)
	if ok, wait := a.aiUserLimits.allow(userID.String()); !ok {
		tooManyRequests(w, r, wait, "ya pediste varios análisis; podrás generar otro en unos minutos")
		return
	}
	if ok, wait := a.aiGlobalLimits.allow(aiGlobalKey); !ok {
		tooManyRequests(w, r, wait, "el análisis con IA está muy solicitado; intenta de nuevo en unos segundos")
		return
	}

	rc := http.NewResponseController(w)
	if err := rc.SetWriteDeadline(time.Now().Add(aiWriteTimeout)); err != nil && !errors.Is(err, http.ErrNotSupported) {
		slog.WarnContext(r.Context(), "ampliar el plazo de escritura del análisis", "error", err)
	}

	report, err := a.Insights.Diagnose(r.Context(), userID)
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, newDiagnosisResponse(report))
}

func newDiagnosisResponse(r usecase.InsightReport) diagnosisResponse {
	resp := diagnosisResponse{
		Headline:        r.Headline,
		Summary:         r.Summary,
		FinanceInsights: nonNil(r.FinanceInsights),
		HabitInsights:   nonNil(r.HabitInsights),
		Suggestions:     make([]suggestionResponse, 0, len(r.Suggestions)),
		Period:          periodResponse{From: r.From.Format(time.DateOnly), To: r.To.Format(time.DateOnly)},
		GeneratedAt:     r.GeneratedAt.UTC(),
	}
	for _, s := range r.Suggestions {
		resp.Suggestions = append(resp.Suggestions, suggestionResponse(s))
	}
	return resp
}

// nonNil devuelve items, o una lista vacía si es nil, para responder [] y no null.
func nonNil(items []string) []string {
	if items == nil {
		return []string{}
	}
	return items
}
