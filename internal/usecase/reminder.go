package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bytebalance/internal/domain/habit"
)

// periodNames nombra el periodo de cada frecuencia en los recordatorios.
var periodNames = map[habit.Frequency]string{
	habit.Daily:   "hoy",
	habit.Weekly:  "esta semana",
	habit.Monthly: "este mes",
}

// ReminderService envía por Telegram los recordatorios de hábitos.
type ReminderService struct {
	reminders habit.ReminderRepository
	logs      habit.LogRepository
	bot       TelegramSender
}

// NewReminderService crea un ReminderService.
func NewReminderService(reminders habit.ReminderRepository, logs habit.LogRepository, bot TelegramSender) *ReminderService {
	return &ReminderService{reminders: reminders, logs: logs, bot: bot}
}

// SendDue envía los recordatorios cuya hora coincide con el minuto de at en la
// zona de cada usuario, salvo los de hábitos que ya cumplieron la meta del
// periodo. Un fallo con un recordatorio no impide enviar los demás: devuelve
// cuántos se enviaron y los errores juntos.
func (s *ReminderService) SendDue(ctx context.Context, at time.Time) (int, error) {
	due, err := s.reminders.ListDue(ctx, at)
	if err != nil {
		return 0, fmt.Errorf("enviar recordatorios: %w", err)
	}
	sent := 0
	var errs []error
	for _, r := range due {
		ok, err := s.send(ctx, r, at)
		if err != nil {
			errs = append(errs, fmt.Errorf("recordatorio del hábito %s: %w", r.Habit.ID, err))
			continue
		}
		if ok {
			sent++
		}
	}
	return sent, errors.Join(errs...)
}

// send envía un recordatorio si corresponde; devuelve false si no hacía falta.
func (s *ReminderService) send(ctx context.Context, r habit.Reminder, at time.Time) (bool, error) {
	loc, err := time.LoadLocation(r.Timezone)
	if err != nil {
		return false, fmt.Errorf("zona horaria: %w", err)
	}
	local := at.In(loc)
	// El repositorio ya filtra por hora; se revalida para no depender de él.
	if rt := r.Habit.ReminderTime; rt == nil || rt.Hour != local.Hour() || rt.Minute != local.Minute() {
		return false, nil
	}

	today := habit.DateOf(local)
	logs, err := s.logs.ListByHabit(ctx, r.Habit.ID, habit.PeriodStart(r.Habit.Frequency, today), today)
	if err != nil {
		return false, fmt.Errorf("leer progreso: %w", err)
	}
	done := 0
	for _, l := range logs {
		done += l.Count
	}
	if done >= r.Habit.TargetCount {
		return false, nil
	}

	if err := s.bot.SendMessage(ctx, r.ChatID, reminderText(r.Habit, done)); err != nil {
		return false, fmt.Errorf("enviar mensaje: %w", err)
	}
	return true, nil
}

// reminderText arma el mensaje. Solo incluye el nombre del hábito y el
// progreso: ningún dato financiero ni personal.
func reminderText(h habit.Habit, done int) string {
	return fmt.Sprintf("Recordatorio: %s\nProgreso %s: %d de %d.", h.Name, periodNames[h.Frequency], done, h.TargetCount)
}
