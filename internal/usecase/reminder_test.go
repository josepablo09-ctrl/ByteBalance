package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/habit"
)

// fakeReminders devuelve siempre los mismos recordatorios (el filtro por hora
// de la base se prueba en el repositorio de PostgreSQL).
type fakeReminders struct {
	due []habit.Reminder
	err error
}

func (f fakeReminders) ListDue(context.Context, time.Time) ([]habit.Reminder, error) {
	return f.due, f.err
}

func TestReminderServiceSendDue(t *testing.T) {
	ctx := context.Background()
	// 13:30 UTC = 07:30 en Costa Rica, viernes 16 de octubre de 2026.
	at := time.Date(2026, 10, 16, 13, 30, 0, 0, time.UTC)
	at730 := &habit.TimeOfDay{Hour: 7, Minute: 30}
	reminder := func(name string, f habit.Frequency, target int, rt *habit.TimeOfDay, chatID int64) habit.Reminder {
		return habit.Reminder{
			Habit:    habit.Habit{ID: domain.NewID(), Name: name, Frequency: f, TargetCount: target, ReminderTime: rt, IsActive: true},
			Timezone: "America/Costa_Rica",
			ChatID:   chatID,
		}
	}
	day := func(d int) time.Time { return time.Date(2026, 10, d, 0, 0, 0, 0, time.UTC) }

	pending := reminder("Meditar", habit.Daily, 1, at730, 1)
	doneToday := reminder("Leer", habit.Daily, 1, at730, 2)
	weeklyHalf := reminder("Correr", habit.Weekly, 3, at730, 3)
	weeklyDone := reminder("Nadar", habit.Weekly, 2, at730, 4)
	otherMinute := reminder("Agua", habit.Daily, 1, &habit.TimeOfDay{Hour: 7, Minute: 31}, 5)

	logs := newFakeLogRepo()
	seed := func(h habit.Reminder, d time.Time, count int) {
		if err := logs.Upsert(ctx, &habit.Log{HabitID: h.Habit.ID, LoggedOn: d, Count: count}); err != nil {
			t.Fatal(err)
		}
	}
	seed(doneToday, day(16), 1)
	seed(weeklyHalf, day(12), 1) // lunes de esta semana
	seed(weeklyHalf, day(9), 5)  // semana anterior: no cuenta
	seed(weeklyDone, day(13), 2)

	bot := &fakeBot{}
	svc := NewReminderService(fakeReminders{due: []habit.Reminder{pending, doneToday, weeklyHalf, weeklyDone, otherMinute}}, logs, bot)

	sent, err := svc.SendDue(ctx, at)
	if err != nil {
		t.Fatalf("SendDue() error inesperado: %v", err)
	}
	if sent != 2 || len(bot.sent) != 2 {
		t.Fatalf("enviados = %d (%v), se esperaban 2: Meditar y Correr", sent, bot.sent)
	}
	want := map[int64]string{
		1: "Recordatorio: Meditar\nProgreso hoy: 0 de 1.",
		3: "Recordatorio: Correr\nProgreso esta semana: 1 de 3.",
	}
	for _, msg := range bot.sent {
		if want[msg.chatID] != msg.text {
			t.Errorf("mensaje al chat %d = %q, se esperaba %q", msg.chatID, msg.text, want[msg.chatID])
		}
	}
}

func TestReminderServiceSendDueContinuesAfterErrors(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2026, 10, 16, 13, 30, 0, 0, time.UTC)
	rt := &habit.TimeOfDay{Hour: 13, Minute: 30}
	good := habit.Reminder{Habit: habit.Habit{ID: domain.NewID(), Name: "A", Frequency: habit.Daily, TargetCount: 1, ReminderTime: rt}, Timezone: "UTC", ChatID: 1}
	badZone := habit.Reminder{Habit: habit.Habit{ID: domain.NewID(), Name: "B", Frequency: habit.Daily, TargetCount: 1, ReminderTime: rt}, Timezone: "Marte/Olympus", ChatID: 2}

	bot := &fakeBot{}
	sent, err := NewReminderService(fakeReminders{due: []habit.Reminder{badZone, good}}, newFakeLogRepo(), bot).SendDue(ctx, at)
	if sent != 1 || len(bot.sent) != 1 || bot.sent[0].chatID != 1 {
		t.Errorf("enviados = %d, %v; se esperaba solo el recordatorio válido", sent, bot.sent)
	}
	if err == nil {
		t.Error("se esperaba el error de la zona horaria inválida")
	}

	failing := NewReminderService(fakeReminders{err: errors.New("base caída")}, newFakeLogRepo(), &fakeBot{})
	if _, err := failing.SendDue(ctx, at); err == nil {
		t.Error("se esperaba el error del repositorio")
	}
}
