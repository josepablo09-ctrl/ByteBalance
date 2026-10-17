package http

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/insight"
	"bytebalance/internal/usecase"
)

// stubInsights simula el caso de uso de análisis y cuenta sus llamadas.
type stubInsights struct {
	report usecase.InsightReport
	err    error
	calls  *int
}

func (s stubInsights) Diagnose(_ context.Context, userID domain.ID) (usecase.InsightReport, error) {
	if s.calls != nil {
		*s.calls++
	}
	if userID != testUserID {
		return usecase.InsightReport{}, fmt.Errorf("usuario inesperado %v", userID)
	}
	return s.report, s.err
}

var sampleReport = usecase.InsightReport{
	Diagnosis: insight.Diagnosis{
		Headline:        "Ahorras el 25 % de tus ingresos",
		Summary:         "Vas bien.",
		FinanceInsights: []string{"Comida es tu mayor gasto"},
		Suggestions:     []insight.Suggestion{{Title: "Tope", Detail: "Fija un tope semanal"}},
	},
	From:        time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC),
	To:          time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC),
	GeneratedAt: time.Date(2026, 10, 16, 14, 30, 0, 0, time.FixedZone("CST", -6*3600)),
}

func TestAIStatus(t *testing.T) {
	for _, tt := range []struct {
		name     string
		insights InsightService
		want     bool
	}{
		{name: "sin llave", want: false},
		{name: "con llave", insights: stubInsights{}, want: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(newTestRouter(Services{Insights: tt.insights}), http.MethodGet, "/api/v1/ai", "", true)
			expectStatus(t, rec, http.StatusOK)
			if got := decodeJSONBody(t, rec)["enabled"]; got != tt.want {
				t.Errorf("enabled = %v, se esperaba %v", got, tt.want)
			}
		})
	}
}

func TestDiagnose(t *testing.T) {
	h := newTestRouter(Services{Insights: stubInsights{report: sampleReport}})

	rec := do(h, http.MethodPost, "/api/v1/ai/diagnose", "", true)

	expectStatus(t, rec, http.StatusOK)
	got := decodeJSONBody(t, rec)
	want := map[string]any{
		"headline":         "Ahorras el 25 % de tus ingresos",
		"summary":          "Vas bien.",
		"finance_insights": []any{"Comida es tu mayor gasto"},
		"habit_insights":   []any{}, // [] y no null
		"suggestions":      []any{map[string]any{"title": "Tope", "detail": "Fija un tope semanal"}},
		"period":           map[string]any{"from": "2026-09-17", "to": "2026-10-16"},
		"generated_at":     "2026-10-16T20:30:00Z",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("respuesta =\n%v\nse esperaba\n%v", got, want)
	}
}

func TestDiagnoseErrors(t *testing.T) {
	tests := []struct {
		name          string
		insights      InsightService
		authenticated bool
		wantStatus    int
		wantCode      string
	}{
		{name: "sin token", insights: stubInsights{}, wantStatus: http.StatusUnauthorized, wantCode: "unauthorized"},
		{name: "sin llave de Gemini", authenticated: true, wantStatus: http.StatusServiceUnavailable, wantCode: "ai_disabled"},
		{
			name:          "sin datos",
			insights:      stubInsights{err: fmt.Errorf("generar diagnóstico: %w", insight.ErrNotEnoughData)},
			authenticated: true, wantStatus: http.StatusUnprocessableEntity, wantCode: "not_enough_data",
		},
		{
			name:          "la IA falla",
			insights:      stubInsights{err: fmt.Errorf("generar diagnóstico: %w: gemini: HTTP 429 RESOURCE_EXHAUSTED", usecase.ErrAIUnavailable)},
			authenticated: true, wantStatus: http.StatusServiceUnavailable, wantCode: "ai_unavailable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(newTestRouter(Services{Insights: tt.insights}), http.MethodPost, "/api/v1/ai/diagnose", "", tt.authenticated)
			expectStatus(t, rec, tt.wantStatus)
			if code := errorCode(t, rec); code != tt.wantCode {
				t.Errorf("code = %q, se esperaba %q", code, tt.wantCode)
			}
			// El detalle del proveedor nunca llega al cliente.
			if body := rec.Body.String(); strings.Contains(body, "RESOURCE_EXHAUSTED") || strings.Contains(body, "gemini") {
				t.Errorf("la respuesta expone detalles internos: %s", body)
			}
		})
	}
}

func TestDiagnoseRateLimitPerUser(t *testing.T) {
	calls := 0
	h := newTestRouter(Services{Insights: stubInsights{report: sampleReport, calls: &calls}})

	for i := range aiUserBurst {
		expectStatus(t, do(h, http.MethodPost, "/api/v1/ai/diagnose", "", true), http.StatusOK)
		if calls != i+1 {
			t.Fatalf("llamadas = %d", calls)
		}
	}
	rec := do(h, http.MethodPost, "/api/v1/ai/diagnose", "", true)
	expectStatus(t, rec, http.StatusTooManyRequests)
	if rec.Header().Get("Retry-After") == "" {
		t.Error("falta la cabecera Retry-After")
	}
	if calls != aiUserBurst {
		t.Errorf("una petición limitada no debe llegar al caso de uso (llamadas = %d)", calls)
	}
	// Consultar el estado no consume análisis.
	expectStatus(t, do(h, http.MethodGet, "/api/v1/ai", "", true), http.StatusOK)
}

func TestDiagnoseRateLimitGlobal(t *testing.T) {
	calls := 0
	a := &api{
		Services:       withStubs(Services{Insights: stubInsights{report: sampleReport, calls: &calls}}),
		aiUserLimits:   newLimiter(100, time.Second, nil),
		aiGlobalLimits: newLimiter(2, time.Hour, nil),
	}
	h := requireAuth(stubTokens{}, http.HandlerFunc(a.diagnose))

	expectStatus(t, do(h, http.MethodPost, "/", "", true), http.StatusOK)
	expectStatus(t, do(h, http.MethodPost, "/", "", true), http.StatusOK)
	rec := do(h, http.MethodPost, "/", "", true)
	expectStatus(t, rec, http.StatusTooManyRequests)
	if code := errorCode(t, rec); code != "too_many_requests" || calls != 2 {
		t.Errorf("code = %q, llamadas = %d", code, calls)
	}
}
