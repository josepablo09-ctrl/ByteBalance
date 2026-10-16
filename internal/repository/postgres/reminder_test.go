package postgres

import (
	"context"
	"slices"
	"testing"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/habit"
)

func TestReminderRepositoryListDue(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)
	users := NewUserRepository(tx)
	habits := NewHabitRepository(tx)

	// Usuario con zona y Telegram; chatID único para no chocar con datos existentes.
	linkedUser := func(timezone string) domain.ID {
		t.Helper()
		u := createTestUser(t, tx)
		u.Timezone = timezone
		if err := u.LinkTelegram(time.Now().UnixNano()); err != nil {
			t.Fatalf("LinkTelegram: %v", err)
		}
		if err := users.Update(ctx, u); err != nil {
			t.Fatalf("actualizar usuario: %v", err)
		}
		return u.ID
	}
	addHabit := func(userID domain.ID, name string, reminder *habit.TimeOfDay) habit.Habit {
		t.Helper()
		h, err := habit.New(habit.Params{UserID: userID, Name: name, ReminderTime: reminder})
		if err != nil {
			t.Fatalf("habit.New: %v", err)
		}
		if err := habits.Create(ctx, h); err != nil {
			t.Fatalf("crear hábito: %v", err)
		}
		return *h
	}
	at730 := &habit.TimeOfDay{Hour: 7, Minute: 30}

	costaRica := linkedUser("America/Costa_Rica")
	tokyo := linkedUser("Asia/Tokyo")
	noTelegram := createTestUser(t, tx).ID

	due := addHabit(costaRica, "Meditar", at730)
	addHabit(costaRica, "Otro minuto", &habit.TimeOfDay{Hour: 7, Minute: 31})
	addHabit(costaRica, "Sin recordatorio", nil)
	paused := addHabit(costaRica, "Pausado", at730)
	paused.Deactivate()
	if err := habits.Update(ctx, &paused); err != nil {
		t.Fatalf("pausar: %v", err)
	}
	deleted := addHabit(costaRica, "Eliminado", at730)
	if err := habits.Delete(ctx, costaRica, deleted.ID); err != nil {
		t.Fatalf("eliminar: %v", err)
	}
	addHabit(tokyo, "Misma hora en Tokio", at730)
	addHabit(noTelegram, "Sin Telegram", at730)

	// 13:30:45 UTC son las 07:30 en Costa Rica (UTC-6) y las 22:30 en Tokio.
	reminders, err := NewReminderRepository(tx).ListDue(ctx, time.Date(2026, 10, 16, 13, 30, 45, 0, time.UTC))
	if err != nil {
		t.Fatalf("ListDue: %v", err)
	}
	// La base puede tener otros datos: solo cuentan los hábitos de este test.
	mine := slices.DeleteFunc(reminders, func(r habit.Reminder) bool {
		return r.Habit.UserID != costaRica && r.Habit.UserID != tokyo && r.Habit.UserID != noTelegram
	})
	if len(mine) != 1 || mine[0].Habit.ID != due.ID {
		names := make([]string, 0, len(mine))
		for _, r := range mine {
			names = append(names, r.Habit.Name)
		}
		t.Fatalf("ListDue() = %v, se esperaba solo %q", names, due.Name)
	}
	if mine[0].Timezone != "America/Costa_Rica" || mine[0].ChatID == 0 || mine[0].Habit.ReminderTime == nil {
		t.Errorf("recordatorio = %+v", mine[0])
	}
}
