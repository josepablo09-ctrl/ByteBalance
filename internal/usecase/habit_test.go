package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/habit"
)

// fakeHabitRepo es un habit.Repository en memoria acotado por usuario.
type fakeHabitRepo struct {
	byID map[domain.ID]habit.Habit
}

var _ habit.Repository = (*fakeHabitRepo)(nil)

func newFakeHabitRepo(habits ...habit.Habit) *fakeHabitRepo {
	f := &fakeHabitRepo{byID: map[domain.ID]habit.Habit{}}
	for _, h := range habits {
		f.byID[h.ID] = h
	}
	return f
}

func (f *fakeHabitRepo) Create(_ context.Context, h *habit.Habit) error {
	h.ID = domain.NewID()
	f.byID[h.ID] = *h
	return nil
}

func (f *fakeHabitRepo) GetByID(_ context.Context, userID, id domain.ID) (*habit.Habit, error) {
	h, ok := f.byID[id]
	if !ok || h.UserID != userID {
		return nil, domain.ErrNotFound
	}
	return &h, nil
}

func (f *fakeHabitRepo) ListByUser(_ context.Context, userID domain.ID, activeOnly bool) ([]habit.Habit, error) {
	var habits []habit.Habit
	for _, h := range f.byID {
		if h.UserID == userID && (h.IsActive || !activeOnly) {
			habits = append(habits, h)
		}
	}
	return habits, nil
}

func (f *fakeHabitRepo) Update(_ context.Context, h *habit.Habit) error {
	if _, err := f.GetByID(context.Background(), h.UserID, h.ID); err != nil {
		return err
	}
	f.byID[h.ID] = *h
	return nil
}

func (f *fakeHabitRepo) Delete(_ context.Context, userID, id domain.ID) error {
	if _, err := f.GetByID(context.Background(), userID, id); err != nil {
		return err
	}
	delete(f.byID, id)
	return nil
}

// fakeLogRepo es un habit.LogRepository en memoria con un registro por hábito y día.
type fakeLogRepo struct {
	byKey     map[string]habit.Log
	upsertErr error
}

var _ habit.LogRepository = (*fakeLogRepo)(nil)

func newFakeLogRepo() *fakeLogRepo {
	return &fakeLogRepo{byKey: map[string]habit.Log{}}
}

func logKey(habitID domain.ID, day time.Time) string {
	return habitID.String() + "/" + habit.DateOf(day).Format(time.DateOnly)
}

func (f *fakeLogRepo) Upsert(_ context.Context, l *habit.Log) error {
	if f.upsertErr != nil {
		return f.upsertErr
	}
	key := logKey(l.HabitID, l.LoggedOn)
	if existing, ok := f.byKey[key]; ok {
		l.ID = existing.ID
	} else {
		l.ID = domain.NewID()
	}
	f.byKey[key] = *l
	return nil
}

func (f *fakeLogRepo) ListByHabit(_ context.Context, habitID domain.ID, from, to time.Time) ([]habit.Log, error) {
	var logs []habit.Log
	for _, l := range f.byKey {
		if l.HabitID == habitID && !l.LoggedOn.Before(habit.DateOf(from)) && !l.LoggedOn.After(habit.DateOf(to)) {
			logs = append(logs, l)
		}
	}
	return logs, nil
}

func (f *fakeLogRepo) Delete(_ context.Context, habitID domain.ID, loggedOn time.Time) error {
	key := logKey(habitID, loggedOn)
	if _, ok := f.byKey[key]; !ok {
		return domain.ErrNotFound
	}
	delete(f.byKey, key)
	return nil
}

func TestHabitServiceCreateHabit(t *testing.T) {
	owner := domain.NewID()
	tests := []struct {
		name    string
		in      CreateHabitInput
		wantErr error
		check   func(t *testing.T, h *habit.Habit)
	}{
		{
			name: "aplica valores por defecto",
			in:   CreateHabitInput{Name: " Leer "},
			check: func(t *testing.T, h *habit.Habit) {
				if h.Name != "Leer" || h.Frequency != habit.Daily || h.TargetCount != 1 || !h.IsActive || h.UserID != owner {
					t.Errorf("hábito creado = %+v", h)
				}
			},
		},
		{
			name: "semanal con recordatorio",
			in:   CreateHabitInput{Name: "Correr", Frequency: habit.Weekly, TargetCount: 3, ReminderTime: &habit.TimeOfDay{Hour: 6, Minute: 30}},
			check: func(t *testing.T, h *habit.Habit) {
				if h.Frequency != habit.Weekly || h.TargetCount != 3 || h.ReminderTime.String() != "06:30" {
					t.Errorf("hábito creado = %+v", h)
				}
			},
		},
		{name: "nombre vacío", in: CreateHabitInput{Name: ""}, wantErr: domain.ErrInvalidInput},
		{name: "frecuencia inválida", in: CreateHabitInput{Name: "Leer", Frequency: "hourly"}, wantErr: domain.ErrInvalidInput},
		{name: "meta negativa", in: CreateHabitInput{Name: "Leer", TargetCount: -2}, wantErr: domain.ErrInvalidInput},
		{name: "recordatorio inválido", in: CreateHabitInput{Name: "Leer", ReminderTime: &habit.TimeOfDay{Hour: 25}}, wantErr: domain.ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			habits := newFakeHabitRepo()
			svc := NewHabitService(habits, newFakeLogRepo(), newFakeUserRepo(), fixedClock)

			h, err := svc.CreateHabit(context.Background(), owner, tt.in)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("CreateHabit() error = %v, se esperaba %v", err, tt.wantErr)
				}
				if len(habits.byID) != 0 {
					t.Error("no debe persistirse un hábito inválido")
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateHabit() error inesperado: %v", err)
			}
			if _, ok := habits.byID[h.ID]; !ok {
				t.Error("el hábito no quedó persistido")
			}
			tt.check(t, h)
		})
	}
}

func TestHabitServiceLogCompletion(t *testing.T) {
	// fixedNow es 2026-10-11 18:30 UTC: en Tokio ya es 12 de octubre y en
	// Costa Rica (UTC-6) todavía es 11 de octubre.
	tokyoUser := existingUser(t, "tokio@example.com", "Asia/Tokyo")
	crUser := existingUser(t, "cr@example.com", "America/Costa_Rica")
	stranger := existingUser(t, "otro@example.com", "UTC")
	active := habit.Habit{ID: domain.NewID(), UserID: crUser.ID, Name: "Meditar", Frequency: habit.Daily, TargetCount: 1, IsActive: true}
	paused := habit.Habit{ID: domain.NewID(), UserID: crUser.ID, Name: "Correr", Frequency: habit.Daily, TargetCount: 1}
	tokyoHabit := habit.Habit{ID: domain.NewID(), UserID: tokyoUser.ID, Name: "Leer", Frequency: habit.Daily, TargetCount: 1, IsActive: true}
	date := func(m time.Month, d int) time.Time { return time.Date(2026, m, d, 0, 0, 0, 0, time.UTC) }
	errDB := errors.New("base caída")

	tests := []struct {
		name      string
		userID    domain.ID
		habitID   domain.ID
		in        LogCompletionInput
		upsertErr error
		wantErr   error
		wantDate  time.Time
		wantCount int
	}{
		{name: "hoy según Costa Rica", userID: crUser.ID, habitID: active.ID, wantDate: date(10, 11), wantCount: 1},
		{name: "hoy según Tokio", userID: tokyoUser.ID, habitID: tokyoHabit.ID, wantDate: date(10, 12), wantCount: 1},
		{name: "día pasado con cantidad y nota", userID: crUser.ID, habitID: active.ID, in: LogCompletionInput{Date: date(10, 1), Count: 3, Note: "doble sesión"}, wantDate: date(10, 1), wantCount: 3},
		{name: "día futuro para el usuario", userID: crUser.ID, habitID: active.ID, in: LogCompletionInput{Date: date(10, 12)}, wantErr: domain.ErrInvalidInput},
		{name: "hábito pausado", userID: crUser.ID, habitID: paused.ID, wantErr: habit.ErrInactive},
		{name: "hábito de otro usuario", userID: stranger.ID, habitID: active.ID, wantErr: domain.ErrNotFound},
		{name: "hábito inexistente", userID: crUser.ID, habitID: domain.NewID(), wantErr: domain.ErrNotFound},
		{name: "cantidad negativa", userID: crUser.ID, habitID: active.ID, in: LogCompletionInput{Count: -1}, wantErr: domain.ErrInvalidInput},
		{name: "error del repositorio", userID: crUser.ID, habitID: active.ID, upsertErr: errDB, wantErr: errDB},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := newFakeLogRepo()
			logs.upsertErr = tt.upsertErr
			svc := NewHabitService(
				newFakeHabitRepo(active, paused, tokyoHabit),
				logs,
				newFakeUserRepo(tokyoUser, crUser, stranger),
				fixedClock,
			)

			l, err := svc.LogCompletion(context.Background(), tt.userID, tt.habitID, tt.in)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("LogCompletion() error = %v, se esperaba %v", err, tt.wantErr)
				}
				if len(logs.byKey) != 0 {
					t.Error("no debe persistirse un registro inválido")
				}
				return
			}
			if err != nil {
				t.Fatalf("LogCompletion() error inesperado: %v", err)
			}
			if !l.LoggedOn.Equal(tt.wantDate) || l.Count != tt.wantCount || l.HabitID != tt.habitID {
				t.Errorf("registro = %+v, se esperaba fecha %v y cantidad %d", l, tt.wantDate, tt.wantCount)
			}
			if _, ok := logs.byKey[logKey(tt.habitID, tt.wantDate)]; !ok {
				t.Error("el registro no quedó persistido")
			}
		})
	}
}

func TestHabitServiceLogCompletionReplacesSameDay(t *testing.T) {
	u := existingUser(t, "ana@example.com", "UTC")
	h := habit.Habit{ID: domain.NewID(), UserID: u.ID, Name: "Agua", Frequency: habit.Daily, TargetCount: 8, IsActive: true}
	logs := newFakeLogRepo()
	svc := NewHabitService(newFakeHabitRepo(h), logs, newFakeUserRepo(u), fixedClock)
	ctx := context.Background()

	first, err := svc.LogCompletion(ctx, u.ID, h.ID, LogCompletionInput{Count: 3})
	if err != nil {
		t.Fatalf("primer LogCompletion() error inesperado: %v", err)
	}
	second, err := svc.LogCompletion(ctx, u.ID, h.ID, LogCompletionInput{Count: 8, Note: "meta cumplida"})
	if err != nil {
		t.Fatalf("segundo LogCompletion() error inesperado: %v", err)
	}
	if second.ID != first.ID || len(logs.byKey) != 1 {
		t.Fatalf("se esperaba reemplazar el registro del día: %d registros", len(logs.byKey))
	}
	if stored := logs.byKey[logKey(h.ID, fixedNow)]; stored.Count != 8 || stored.Note != "meta cumplida" {
		t.Errorf("registro guardado = %+v", stored)
	}
}

func TestHabitServiceListHabits(t *testing.T) {
	owner := existingUser(t, "ana@example.com", "UTC")
	other := existingUser(t, "otro@example.com", "UTC")
	active := habit.Habit{ID: domain.NewID(), UserID: owner.ID, Name: "Leer", IsActive: true}
	paused := habit.Habit{ID: domain.NewID(), UserID: owner.ID, Name: "Correr"}
	foreign := habit.Habit{ID: domain.NewID(), UserID: other.ID, Name: "Ajeno", IsActive: true}
	svc := NewHabitService(newFakeHabitRepo(active, paused, foreign), newFakeLogRepo(), newFakeUserRepo(owner, other), fixedClock)

	tests := []struct {
		name       string
		activeOnly bool
		want       []domain.ID
	}{
		{name: "todos", want: []domain.ID{active.ID, paused.ID}},
		{name: "solo activos", activeOnly: true, want: []domain.ID{active.ID}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			habits, err := svc.ListHabits(context.Background(), owner.ID, tt.activeOnly)
			if err != nil {
				t.Fatalf("ListHabits() error inesperado: %v", err)
			}
			got := make(map[domain.ID]bool, len(habits))
			for _, h := range habits {
				got[h.ID] = true
			}
			if len(got) != len(tt.want) {
				t.Fatalf("ListHabits() = %d hábitos, se esperaban %d", len(got), len(tt.want))
			}
			for _, id := range tt.want {
				if !got[id] {
					t.Errorf("falta el hábito %v", id)
				}
			}
		})
	}
}

func TestHabitServiceListLogs(t *testing.T) {
	// fixedNow es 2026-10-11 18:30 UTC: en Tokio ya es 12 de octubre.
	owner := existingUser(t, "tokio@example.com", "Asia/Tokyo")
	stranger := existingUser(t, "otro@example.com", "UTC")
	h := habit.Habit{ID: domain.NewID(), UserID: owner.ID, Name: "Leer", Frequency: habit.Daily, TargetCount: 1, IsActive: true}
	date := func(m time.Month, d int) time.Time { return time.Date(2026, m, d, 0, 0, 0, 0, time.UTC) }

	logs := newFakeLogRepo()
	for _, d := range []time.Time{date(9, 12), date(9, 13), date(10, 1), date(10, 12)} {
		if err := logs.Upsert(context.Background(), &habit.Log{HabitID: h.ID, LoggedOn: d, Count: 1}); err != nil {
			t.Fatalf("sembrar registro: %v", err)
		}
	}
	svc := NewHabitService(newFakeHabitRepo(h), logs, newFakeUserRepo(owner, stranger), fixedClock)

	tests := []struct {
		name      string
		userID    domain.ID
		from, to  time.Time
		wantDates []time.Time
		wantErr   error
	}{
		{name: "por defecto los últimos 30 días del usuario", userID: owner.ID, wantDates: []time.Time{date(9, 13), date(10, 1), date(10, 12)}},
		{name: "rango explícito inclusivo", userID: owner.ID, from: date(9, 12), to: date(10, 1), wantDates: []time.Time{date(9, 12), date(9, 13), date(10, 1)}},
		{name: "solo desde", userID: owner.ID, from: date(10, 1), wantDates: []time.Time{date(10, 1), date(10, 12)}},
		{name: "solo hasta", userID: owner.ID, to: date(10, 1), wantDates: []time.Time{date(9, 12), date(9, 13), date(10, 1)}},
		{name: "rango máximo", userID: owner.ID, from: date(10, 1).AddDate(0, 0, -365), to: date(10, 1), wantDates: []time.Time{date(9, 12), date(9, 13), date(10, 1)}},
		{name: "rango excesivo", userID: owner.ID, from: date(10, 1).AddDate(0, 0, -366), to: date(10, 1), wantErr: domain.ErrInvalidInput},
		{name: "desde posterior a hasta", userID: owner.ID, from: date(10, 2), to: date(10, 1), wantErr: domain.ErrInvalidInput},
		{name: "hábito de otro usuario", userID: stranger.ID, wantErr: domain.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.ListLogs(context.Background(), tt.userID, h.ID, tt.from, tt.to)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ListLogs() error = %v, se esperaba %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ListLogs() error inesperado: %v", err)
			}
			gotDates := make(map[time.Time]bool, len(got))
			for _, l := range got {
				gotDates[l.LoggedOn] = true
			}
			if len(gotDates) != len(tt.wantDates) {
				t.Fatalf("ListLogs() = %v, se esperaban las fechas %v", got, tt.wantDates)
			}
			for _, d := range tt.wantDates {
				if !gotDates[d] {
					t.Errorf("falta el registro del %s", d.Format(time.DateOnly))
				}
			}
		})
	}
}
