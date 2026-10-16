package postgres

import (
	"context"
	"fmt"
	"time"

	"bytebalance/internal/domain/habit"
)

// ReminderRepository implementa habit.ReminderRepository sobre PostgreSQL.
type ReminderRepository struct {
	db DBTX
}

var _ habit.ReminderRepository = (*ReminderRepository)(nil)

// NewReminderRepository crea un ReminderRepository.
func NewReminderRepository(db DBTX) *ReminderRepository {
	return &ReminderRepository{db: db}
}

// ListDue compara la hora del recordatorio con la hora local de cada usuario
// en SQL, así solo se leen los hábitos que tocan en este minuto.
func (r *ReminderRepository) ListDue(ctx context.Context, at time.Time) ([]habit.Reminder, error) {
	const q = `SELECT h.id, h.user_id, h.name, COALESCE(h.description, ''), h.frequency, h.target_count,
			h.reminder_time, h.is_active, h.created_at, h.updated_at, h.deleted_at,
			u.timezone, u.telegram_chat_id
		FROM habits h
		JOIN users u ON u.id = h.user_id
		WHERE h.deleted_at IS NULL AND h.is_active AND h.reminder_time IS NOT NULL
			AND u.deleted_at IS NULL AND u.telegram_chat_id IS NOT NULL
			AND h.reminder_time = date_trunc('minute', $1::timestamptz AT TIME ZONE u.timezone)::time
		ORDER BY h.id`
	rows, err := r.db.Query(ctx, q, at)
	if err != nil {
		return nil, fmt.Errorf("listar recordatorios: %w", mapError(err))
	}
	defer rows.Close()

	var reminders []habit.Reminder
	for rows.Next() {
		var rem habit.Reminder
		h, err := scanHabitWith(rows, &rem.Timezone, &rem.ChatID)
		if err != nil {
			return nil, fmt.Errorf("listar recordatorios: %w", err)
		}
		rem.Habit = h
		reminders = append(reminders, rem)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listar recordatorios: %w", mapError(err))
	}
	return reminders, nil
}
