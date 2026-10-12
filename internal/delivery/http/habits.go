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
	writeJSON(w, http.StatusOK, habitLogResponse{
		ID:      l.ID,
		HabitID: l.HabitID,
		Date:    l.LoggedOn.Format(time.DateOnly),
		Count:   l.Count,
		Note:    l.Note,
	})
}
