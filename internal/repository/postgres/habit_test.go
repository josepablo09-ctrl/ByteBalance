package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/habit"
)

func TestHabitRepository(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)
	repo := NewHabitRepository(tx)
	owner := createTestUser(t, tx)
	other := createTestUser(t, tx)

	newHabit := func(p habit.Params) *habit.Habit {
		t.Helper()
		p.UserID = owner.ID
		h, err := habit.New(p)
		if err != nil {
			t.Fatalf("habit.New: %v", err)
		}
		if err := repo.Create(ctx, h); err != nil {
			t.Fatalf("Create: %v", err)
		}
		return h
	}
	reading := newHabit(habit.Params{Name: "Leer", Description: "20 páginas", ReminderTime: &habit.TimeOfDay{Hour: 21, Minute: 45}})
	running := newHabit(habit.Params{Name: "Correr", Frequency: habit.Weekly, TargetCount: 3})

	t.Run("campos opcionales", func(t *testing.T) {
		tests := []struct {
			name         string
			id           domain.ID
			wantReminder *habit.TimeOfDay
			wantDesc     string
		}{
			{name: "con recordatorio y descripción", id: reading.ID, wantReminder: &habit.TimeOfDay{Hour: 21, Minute: 45}, wantDesc: "20 páginas"},
			{name: "sin recordatorio ni descripción", id: running.ID},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := repo.GetByID(ctx, owner.ID, tt.id)
				if err != nil {
					t.Fatalf("GetByID: %v", err)
				}
				if got.Description != tt.wantDesc {
					t.Errorf("Description = %q, se esperaba %q", got.Description, tt.wantDesc)
				}
				switch {
				case tt.wantReminder == nil && got.ReminderTime != nil:
					t.Errorf("ReminderTime = %v, se esperaba nil", got.ReminderTime)
				case tt.wantReminder != nil && (got.ReminderTime == nil || *got.ReminderTime != *tt.wantReminder):
					t.Errorf("ReminderTime = %v, se esperaba %v", got.ReminderTime, tt.wantReminder)
				}
			})
		}
	})

	t.Run("update y list solo activos", func(t *testing.T) {
		running.Deactivate()
		running.TargetCount = 4
		if err := repo.Update(ctx, running); err != nil {
			t.Fatalf("Update: %v", err)
		}
		tests := []struct {
			name       string
			activeOnly bool
			want       []domain.ID
		}{
			{name: "todos", want: []domain.ID{reading.ID, running.ID}},
			{name: "solo activos", activeOnly: true, want: []domain.ID{reading.ID}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := repo.ListByUser(ctx, owner.ID, tt.activeOnly)
				if err != nil {
					t.Fatalf("ListByUser: %v", err)
				}
				if len(got) != len(tt.want) {
					t.Fatalf("ListByUser devolvió %d hábitos, se esperaban %d", len(got), len(tt.want))
				}
				for i := range got {
					if got[i].ID != tt.want[i] {
						t.Errorf("posición %d = %v, se esperaba %v", i, got[i].ID, tt.want[i])
					}
				}
			})
		}
	})

	t.Run("otro usuario no lo ve", func(t *testing.T) {
		if _, err := repo.GetByID(ctx, other.ID, reading.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("GetByID error = %v, se esperaba ErrNotFound", err)
		}
	})

	t.Run("delete", func(t *testing.T) {
		if err := repo.Delete(ctx, owner.ID, running.ID); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if _, err := repo.GetByID(ctx, owner.ID, running.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("GetByID tras Delete error = %v, se esperaba ErrNotFound", err)
		}
	})
}

func TestHabitLogRepository(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)
	repo := NewHabitLogRepository(tx)
	owner := createTestUser(t, tx)

	h, err := habit.New(habit.Params{UserID: owner.ID, Name: "Meditar"})
	if err != nil {
		t.Fatalf("habit.New: %v", err)
	}
	if err := NewHabitRepository(tx).Create(ctx, h); err != nil {
		t.Fatalf("crear hábito: %v", err)
	}

	costaRica := time.FixedZone("CST", -6*60*60)
	day := func(d int) time.Time { return time.Date(2026, 10, d, 23, 30, 0, 0, costaRica) }
	record := func(d, count int, note string) *habit.Log {
		t.Helper()
		l, err := h.Record(day(d), count, note)
		if err != nil {
			t.Fatalf("Record: %v", err)
		}
		if err := repo.Upsert(ctx, l); err != nil {
			t.Fatalf("Upsert: %v", err)
		}
		return l
	}

	first := record(1, 1, "")
	record(3, 2, "")
	record(7, 1, "")

	t.Run("upsert del mismo día reemplaza", func(t *testing.T) {
		again := record(1, 5, "mañana y tarde")
		if again.ID != first.ID {
			t.Errorf("Upsert creó otro registro (%v) en lugar de reemplazar %v", again.ID, first.ID)
		}
		logs, err := repo.ListByHabit(ctx, h.ID, day(1), day(1))
		if err != nil || len(logs) != 1 {
			t.Fatalf("ListByHabit = %+v, %v", logs, err)
		}
		if logs[0].Count != 5 || logs[0].Note != "mañana y tarde" {
			t.Errorf("registro tras Upsert = %+v", logs[0])
		}
		// 23:30 en Costa Rica: debe guardarse el día local, no el día UTC.
		if want := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC); !logs[0].LoggedOn.Equal(want) {
			t.Errorf("LoggedOn = %v, se esperaba %v", logs[0].LoggedOn, want)
		}
	})

	t.Run("list por rango inclusivo", func(t *testing.T) {
		tests := []struct {
			name     string
			from, to int
			wantDays []int
		}{
			{name: "todo", from: 1, to: 31, wantDays: []int{1, 3, 7}},
			{name: "extremos incluidos", from: 3, to: 7, wantDays: []int{3, 7}},
			{name: "vacío", from: 8, to: 9},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				logs, err := repo.ListByHabit(ctx, h.ID, day(tt.from), day(tt.to))
				if err != nil {
					t.Fatalf("ListByHabit: %v", err)
				}
				if len(logs) != len(tt.wantDays) {
					t.Fatalf("ListByHabit devolvió %d registros, se esperaban %d", len(logs), len(tt.wantDays))
				}
				for i, l := range logs {
					if l.LoggedOn.Day() != tt.wantDays[i] {
						t.Errorf("posición %d = día %d, se esperaba %d", i, l.LoggedOn.Day(), tt.wantDays[i])
					}
				}
			})
		}
	})

	t.Run("hábito inexistente", func(t *testing.T) {
		expectErrInSavepoint(t, tx, domain.ErrNotFound, func(db DBTX) error {
			l, err := habit.NewLog(domain.NewID(), day(1), 1, "")
			if err != nil {
				return err
			}
			return NewHabitLogRepository(db).Upsert(ctx, l)
		})
	})

	t.Run("delete", func(t *testing.T) {
		if err := repo.Delete(ctx, h.ID, day(3)); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if err := repo.Delete(ctx, h.ID, day(3)); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("segundo Delete error = %v, se esperaba ErrNotFound", err)
		}
	})
}
