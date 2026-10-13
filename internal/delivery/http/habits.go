package http

import (
	"net/http"
	"strings"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/habit"
	"bytebalance/internal/usecase"
)

type createHabitRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Frequency   string `json:"frequency"`
	TargetCount int    `json:"target_count"`
	// ReminderTime tiene formato HH:MM en la zona horaria del usuario.
	ReminderTime *string `json:"reminder_time"`
}

type habitResponse struct {
	ID           domain.ID       `json:"id"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Frequency    habit.Frequency `json:"frequency"`
	TargetCount  int             `json:"target_count"`
	ReminderTime *string         `json:"reminder_time"`
	IsActive     bool            `json:"is_active"`
	CreatedAt    time.Time       `json:"created_at"`
}

func newHabitResponse(h *habit.Habit) habitResponse {
	resp := habitResponse{
		ID:          h.ID,
		Name:        h.Name,
		Description: h.Description,
		Frequency:   h.Frequency,
		TargetCount: h.TargetCount,
		IsActive:    h.IsActive,
		CreatedAt:   h.CreatedAt,
	}
	if h.ReminderTime != nil {
		s := h.ReminderTime.String()
		resp.ReminderTime = &s
	}
	return resp
}

type logCompletionRequest struct {
	// Date tiene formato YYYY-MM-DD; ausente equivale a hoy para el usuario.
	Date  *string `json:"date"`
	Count int     `json:"count"`
	Note  string  `json:"note"`
}

type habitLogResponse struct {
	ID      domain.ID `json:"id"`
	HabitID domain.ID `json:"habit_id"`
	Date    string    `json:"date"`
	Count   int       `json:"count"`
	Note    string    `json:"note"`
}

func newHabitLogResponse(l *habit.Log) habitLogResponse {
	return habitLogResponse{
		ID:      l.ID,
		HabitID: l.HabitID,
		Date:    l.LoggedOn.Format(time.DateOnly),
		Count:   l.Count,
		Note:    l.Note,
	}
}

// listHabits atiende GET /api/v1/habits. Con ?active=true omite los pausados.
func (a *api) listHabits(w http.ResponseWriter, r *http.Request) {
	activeOnly, err := queryBool(r.URL.Query(), "active")
	if err != nil {
		respondError(w, r, err)
		return
	}
	habits, err := a.Habits.ListHabits(r.Context(), userIDFrom(r), activeOnly)
	if err != nil {
		respondError(w, r, err)
		return
	}
	resp := make([]habitResponse, 0, len(habits))
	for i := range habits {
		resp = append(resp, newHabitResponse(&habits[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"habits": resp})
}

// listLogs atiende GET /api/v1/habits/{id}/logs?from=YYYY-MM-DD&to=YYYY-MM-DD.
// Sin fechas devuelve los últimos 30 días.
func (a *api) listLogs(w http.ResponseWriter, r *http.Request) {
	habitID, err := pathID(r, "id")
	if err != nil {
		respondError(w, r, err)
		return
	}
	q := r.URL.Query()
	from, err := queryDate(q, "from")
	if err != nil {
		respondError(w, r, err)
		return
	}
	to, err := queryDate(q, "to")
	if err != nil {
		respondError(w, r, err)
		return
	}

	logs, err := a.Habits.ListLogs(r.Context(), userIDFrom(r), habitID, from, to)
	if err != nil {
		respondError(w, r, err)
		return
	}
	resp := make([]habitLogResponse, 0, len(logs))
	for i := range logs {
		resp = append(resp, newHabitLogResponse(&logs[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"logs": resp})
}

// createHabit atiende POST /api/v1/habits.
func (a *api) createHabit(w http.ResponseWriter, r *http.Request) {
	var req createHabitRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, r, err)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		respondError(w, r, badRequest("name es obligatorio"))
		return
	}
	in := usecase.CreateHabitInput{
		Name:        req.Name,
		Description: req.Description,
		Frequency:   habit.Frequency(req.Frequency),
		TargetCount: req.TargetCount,
	}
	if req.ReminderTime != nil {
		t, err := time.Parse("15:04", *req.ReminderTime)
		if err != nil {
			respondError(w, r, badRequest("reminder_time debe tener formato HH:MM"))
			return
		}
		in.ReminderTime = &habit.TimeOfDay{Hour: t.Hour(), Minute: t.Minute()}
	}

	h, err := a.Habits.CreateHabit(r.Context(), userIDFrom(r), in)
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, newHabitResponse(h))
}

// logCompletion atiende POST /api/v1/habits/{id}/logs. El cuerpo es opcional:
// sin él se registra una vez el día de hoy.
func (a *api) logCompletion(w http.ResponseWriter, r *http.Request) {
	habitID, err := pathID(r, "id")
	if err != nil {
		respondError(w, r, err)
		return
	}
	var req logCompletionRequest
	if err := decodeOptionalJSON(w, r, &req); err != nil {
		respondError(w, r, err)
		return
	}
	in := usecase.LogCompletionInput{Count: req.Count, Note: req.Note}
	if req.Date != nil {
		d, err := time.Parse(time.DateOnly, *req.Date)
		if err != nil {
			respondError(w, r, badRequest("date debe tener formato YYYY-MM-DD"))
			return
		}
		in.Date = d
	}

	l, err := a.Habits.LogCompletion(r.Context(), userIDFrom(r), habitID, in)
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, newHabitLogResponse(l))
}
