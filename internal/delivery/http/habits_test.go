package http

import (
	"fmt"
	"net/http"
	"strings"
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

func TestListHabits(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		wantStatus     int
		wantActiveOnly bool
	}{
		{name: "todos", wantStatus: http.StatusOK},
		{name: "solo activos", query: "?active=true", wantStatus: http.StatusOK, wantActiveOnly: true},
		{name: "active inválido", query: "?active=quizas", wantStatus: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listFn := func(id domain.ID, activeOnly bool) ([]habit.Habit, error) {
				if id != testUserID || activeOnly != tt.wantActiveOnly {
					t.Errorf("ListHabits(%v, %v)", id, activeOnly)
				}
				return []habit.Habit{
					{ID: domain.NewID(), UserID: id, Name: "Leer", Frequency: habit.Daily, TargetCount: 1, IsActive: true},
					{ID: domain.NewID(), UserID: id, Name: "Correr", Frequency: habit.Weekly, TargetCount: 3, ReminderTime: &habit.TimeOfDay{Hour: 6, Minute: 30}},
				}, nil
			}
			h := newTestRouter(Services{Habits: stubHabits{listHabits: listFn}})
			rec := do(h, http.MethodGet, "/api/v1/habits"+tt.query, "", true)
			expectStatus(t, rec, tt.wantStatus)
			if tt.wantStatus != http.StatusOK {
				return
			}
			habits, ok := decodeJSONBody(t, rec)["habits"].([]any)
			if !ok || len(habits) != 2 {
				t.Fatalf("habits = %v", rec.Body.String())
			}
			if second := habits[1].(map[string]any); second["reminder_time"] != "06:30" || second["is_active"] != false {
				t.Errorf("segundo hábito = %v", second)
			}
		})
	}
}

func TestListHabitsEmptyIsArray(t *testing.T) {
	h := newTestRouter(Services{Habits: stubHabits{listHabits: func(domain.ID, bool) ([]habit.Habit, error) { return nil, nil }}})
	rec := do(h, http.MethodGet, "/api/v1/habits", "", true)
	expectStatus(t, rec, http.StatusOK)
	if got := strings.TrimSpace(rec.Body.String()); got != `{"habits":[]}` {
		t.Errorf("body = %s, se esperaba una lista vacía", got)
	}
}

func TestListLogs(t *testing.T) {
	habitID := domain.NewID()
	path := "/api/v1/habits/" + habitID.String() + "/logs"
	date := func(m time.Month, d int) time.Time { return time.Date(2026, m, d, 0, 0, 0, 0, time.UTC) }

	tests := []struct {
		name       string
		target     string
		listFn     func(domain.ID, domain.ID, time.Time, time.Time) ([]habit.Log, error)
		wantStatus int
	}{
		{
			name:   "rango explícito",
			target: path + "?from=2026-10-01&to=2026-10-11",
			listFn: func(userID, gotHabit domain.ID, from, to time.Time) ([]habit.Log, error) {
				if userID != testUserID || gotHabit != habitID || !from.Equal(date(10, 1)) || !to.Equal(date(10, 11)) {
					t.Errorf("ListLogs(%v, %v, %v, %v)", userID, gotHabit, from, to)
				}
				return []habit.Log{
					{ID: domain.NewID(), HabitID: habitID, LoggedOn: date(10, 1), Count: 1},
					{ID: domain.NewID(), HabitID: habitID, LoggedOn: date(10, 5), Count: 2, Note: "doble"},
				}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "sin fechas delega los valores por defecto",
			target: path,
			listFn: func(_, _ domain.ID, from, to time.Time) ([]habit.Log, error) {
				if !from.IsZero() || !to.IsZero() {
					t.Errorf("se esperaban fechas cero, llegaron %v y %v", from, to)
				}
				return nil, nil
			},
			wantStatus: http.StatusOK,
		},
		{name: "id inválido", target: "/api/v1/habits/no-uuid/logs", wantStatus: http.StatusBadRequest},
		{name: "fecha inválida", target: path + "?from=01/10/2026", wantStatus: http.StatusBadRequest},
		{
			name:   "rango inválido",
			target: path + "?from=2026-10-11&to=2026-10-01",
			listFn: func(domain.ID, domain.ID, time.Time, time.Time) ([]habit.Log, error) {
				return nil, fmt.Errorf("listar registros: %w: la fecha inicial es posterior a la final", domain.ErrInvalidInput)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:   "hábito de otro usuario",
			target: path,
			listFn: func(domain.ID, domain.ID, time.Time, time.Time) ([]habit.Log, error) {
				return nil, fmt.Errorf("listar registros: %w", domain.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestRouter(Services{Habits: stubHabits{listLogs: tt.listFn}})
			rec := do(h, http.MethodGet, tt.target, "", true)
			expectStatus(t, rec, tt.wantStatus)
			if tt.wantStatus != http.StatusOK {
				return
			}
			logs, ok := decodeJSONBody(t, rec)["logs"].([]any)
			if !ok {
				t.Fatalf("logs no es una lista: %s", rec.Body.String())
			}
			if tt.listFn != nil && len(logs) == 2 {
				if second := logs[1].(map[string]any); second["date"] != "2026-10-05" || second["note"] != "doble" {
					t.Errorf("segundo registro = %v", second)
				}
			}
		})
	}
}

func TestUpdateHabit(t *testing.T) {
	habitID := domain.NewID()
	path := "/api/v1/habits/" + habitID.String()
	echo := func(check func(usecase.UpdateHabitInput)) func(domain.ID, domain.ID, usecase.UpdateHabitInput) (*habit.Habit, error) {
		return func(userID, gotHabit domain.ID, in usecase.UpdateHabitInput) (*habit.Habit, error) {
			if userID != testUserID || gotHabit != habitID {
				t.Errorf("UpdateHabit(%v, %v)", userID, gotHabit)
			}
			check(in)
			h := &habit.Habit{ID: habitID, Name: "Leer", Frequency: habit.Daily, TargetCount: 1, IsActive: true, ReminderTime: in.ReminderTime}
			if in.IsActive != nil {
				h.IsActive = *in.IsActive
			}
			return h, nil
		}
	}

	tests := []struct {
		name       string
		target     string
		body       string
		updateFn   func(domain.ID, domain.ID, usecase.UpdateHabitInput) (*habit.Habit, error)
		wantStatus int
		wantBody   map[string]any
	}{
		{
			name:   "pausa",
			target: path,
			body:   `{"is_active":false}`,
			updateFn: echo(func(in usecase.UpdateHabitInput) {
				if in.IsActive == nil || *in.IsActive || in.SetReminder || in.Name != nil {
					t.Errorf("UpdateHabitInput = %+v", in)
				}
			}),
			wantStatus: http.StatusOK,
			wantBody:   map[string]any{"is_active": false},
		},
		{
			name:   "cambia el recordatorio",
			target: path,
			body:   `{"reminder_time":"21:30","target_count":2,"frequency":"weekly"}`,
			updateFn: echo(func(in usecase.UpdateHabitInput) {
				if !in.SetReminder || in.ReminderTime == nil || *in.ReminderTime != (habit.TimeOfDay{Hour: 21, Minute: 30}) {
					t.Errorf("recordatorio = %+v", in)
				}
				if in.TargetCount == nil || *in.TargetCount != 2 || in.Frequency == nil || *in.Frequency != habit.Weekly {
					t.Errorf("UpdateHabitInput = %+v", in)
				}
			}),
			wantStatus: http.StatusOK,
			wantBody:   map[string]any{"reminder_time": "21:30"},
		},
		{
			name:   "null quita el recordatorio",
			target: path,
			body:   `{"reminder_time":null}`,
			updateFn: echo(func(in usecase.UpdateHabitInput) {
				if !in.SetReminder || in.ReminderTime != nil {
					t.Errorf("UpdateHabitInput = %+v", in)
				}
			}),
			wantStatus: http.StatusOK,
			wantBody:   map[string]any{"reminder_time": nil},
		},
		{name: "sin campos", target: path, body: `{}`, wantStatus: http.StatusBadRequest},
		{name: "hora inválida", target: path, body: `{"reminder_time":"7pm"}`, wantStatus: http.StatusBadRequest},
		{name: "recordatorio no textual", target: path, body: `{"reminder_time":730}`, wantStatus: http.StatusBadRequest},
		{name: "campo desconocido", target: path, body: `{"color":"rojo"}`, wantStatus: http.StatusBadRequest},
		{name: "id inválido", target: "/api/v1/habits/no-uuid", body: `{"is_active":true}`, wantStatus: http.StatusBadRequest},
		{
			name:   "valor inválido para el dominio",
			target: path,
			body:   `{"target_count":0}`,
			updateFn: func(domain.ID, domain.ID, usecase.UpdateHabitInput) (*habit.Habit, error) {
				return nil, fmt.Errorf("actualizar hábito: %w: la meta debe ser mayor que cero", domain.ErrInvalidInput)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:   "hábito de otro usuario",
			target: path,
			body:   `{"is_active":true}`,
			updateFn: func(domain.ID, domain.ID, usecase.UpdateHabitInput) (*habit.Habit, error) {
				return nil, fmt.Errorf("actualizar hábito: %w", domain.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestRouter(Services{Habits: stubHabits{updateHabit: tt.updateFn}})
			rec := do(h, http.MethodPatch, tt.target, tt.body, true)
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

func TestDeleteHabit(t *testing.T) {
	habitID := domain.NewID()
	tests := []struct {
		name       string
		target     string
		deleteErr  error
		wantStatus int
	}{
		{name: "elimina", target: "/api/v1/habits/" + habitID.String(), wantStatus: http.StatusNoContent},
		{name: "inexistente", target: "/api/v1/habits/" + habitID.String(), deleteErr: fmt.Errorf("eliminar hábito: %w", domain.ErrNotFound), wantStatus: http.StatusNotFound},
		{name: "id inválido", target: "/api/v1/habits/no-uuid", wantStatus: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestRouter(Services{Habits: stubHabits{deleteHabit: func(userID, gotHabit domain.ID) error {
				if userID != testUserID || gotHabit != habitID {
					t.Errorf("DeleteHabit(%v, %v)", userID, gotHabit)
				}
				return tt.deleteErr
			}}})
			rec := do(h, http.MethodDelete, tt.target, "", true)
			expectStatus(t, rec, tt.wantStatus)
			if tt.wantStatus == http.StatusNoContent && rec.Body.Len() != 0 {
				t.Errorf("un 204 no debe tener cuerpo: %q", rec.Body.String())
			}
		})
	}
}
