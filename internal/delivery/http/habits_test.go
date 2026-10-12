package http

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/habit"
	"bytebalance/internal/usecase"
)

func TestCreateHabit(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		createFn   func(domain.ID, usecase.CreateHabitInput) (*habit.Habit, error)
		wantStatus int
		wantBody   map[string]any
	}{
		{
			name: "con recordatorio",
			body: `{"name":"Correr","frequency":"weekly","target_count":3,"reminder_time":"06:30"}`,
			createFn: func(id domain.ID, in usecase.CreateHabitInput) (*habit.Habit, error) {
				if id != testUserID || in.ReminderTime == nil || *in.ReminderTime != (habit.TimeOfDay{Hour: 6, Minute: 30}) {
					t.Errorf("CreateHabit(%v, %+v)", id, in)
				}
				return &habit.Habit{ID: domain.NewID(), UserID: id, Name: in.Name, Frequency: in.Frequency, TargetCount: 3, ReminderTime: in.ReminderTime, IsActive: true}, nil
			},
			wantStatus: http.StatusCreated,
			wantBody:   map[string]any{"reminder_time": "06:30", "frequency": "weekly", "is_active": true},
		},
		{
			name: "sin recordatorio",
			body: `{"name":"Leer"}`,
			createFn: func(id domain.ID, in usecase.CreateHabitInput) (*habit.Habit, error) {
				return &habit.Habit{ID: domain.NewID(), UserID: id, Name: in.Name, Frequency: habit.Daily, TargetCount: 1, IsActive: true}, nil
			},
			wantStatus: http.StatusCreated,
			wantBody:   map[string]any{"reminder_time": nil},
		},
		{name: "falta name", body: `{"frequency":"daily"}`, wantStatus: http.StatusBadRequest},
		{name: "hora inválida", body: `{"name":"Leer","reminder_time":"25:00"}`, wantStatus: http.StatusBadRequest},
		{name: "hora con segundos", body: `{"name":"Leer","reminder_time":"06:30:00"}`, wantStatus: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestRouter(Services{Habits: stubHabits{createHabit: tt.createFn}})
			rec := do(h, http.MethodPost, "/api/v1/habits", tt.body, true)
			expectStatus(t, rec, tt.wantStatus)
			if tt.wantBody == nil {
				return
			}
			body := decodeJSONBody(t, rec)
			for k, v := range tt.wantBody {
				if body[k] != v {
					t.Errorf("%s = %v, se esperaba %v", k, body[k], v)
				}
			}
		})
	}
}

func TestLogCompletion(t *testing.T) {
	habitID := domain.NewID()
	path := "/api/v1/habits/" + habitID.String() + "/logs"
	okLog := func(in usecase.LogCompletionInput) (*habit.Log, error) {
		date := in.Date
		if date.IsZero() {
			date = time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
		}
		return &habit.Log{ID: domain.NewID(), HabitID: habitID, LoggedOn: date, Count: max(in.Count, 1), Note: in.Note}, nil
	}

	tests := []struct {
		name       string
		path       string
		body       string
		logFn      func(domain.ID, domain.ID, usecase.LogCompletionInput) (*habit.Log, error)
		wantStatus int
		wantDate   string
	}{
		{
			name: "sin cuerpo registra hoy",
			path: path,
			logFn: func(userID, gotHabit domain.ID, in usecase.LogCompletionInput) (*habit.Log, error) {
				if userID != testUserID || gotHabit != habitID || !in.Date.IsZero() || in.Count != 0 {
					t.Errorf("LogCompletion(%v, %v, %+v)", userID, gotHabit, in)
				}
				return okLog(in)
			},
			wantStatus: http.StatusOK,
			wantDate:   "2026-10-11",
		},
		{
			name: "día pasado con cantidad",
			path: path,
			body: `{"date":"2026-10-01","count":3,"note":"doble"}`,
			logFn: func(_, _ domain.ID, in usecase.LogCompletionInput) (*habit.Log, error) {
				if in.Count != 3 || in.Note != "doble" {
					t.Errorf("LogCompletionInput = %+v", in)
				}
				return okLog(in)
			},
			wantStatus: http.StatusOK,
			wantDate:   "2026-10-01",
		},
		{name: "id inválido", path: "/api/v1/habits/no-uuid/logs", wantStatus: http.StatusBadRequest},
		{name: "fecha inválida", path: path, body: `{"date":"01/10/2026"}`, wantStatus: http.StatusBadRequest},
		{
			name: "hábito pausado",
			path: path,
			logFn: func(domain.ID, domain.ID, usecase.LogCompletionInput) (*habit.Log, error) {
				return nil, fmt.Errorf("registrar cumplimiento: %w", habit.ErrInactive)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "hábito de otro usuario",
			path: path,
			logFn: func(domain.ID, domain.ID, usecase.LogCompletionInput) (*habit.Log, error) {
				return nil, fmt.Errorf("registrar cumplimiento: %w", domain.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestRouter(Services{Habits: stubHabits{logCompletion: tt.logFn}})
			rec := do(h, http.MethodPost, tt.path, tt.body, true)
			expectStatus(t, rec, tt.wantStatus)
			if tt.wantDate != "" {
				if body := decodeJSONBody(t, rec); body["date"] != tt.wantDate {
					t.Errorf("date = %v, se esperaba %s", body["date"], tt.wantDate)
				}
			}
		})
	}
}
