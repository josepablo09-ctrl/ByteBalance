package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/habit"
)

// HabitLogRepository implementa habit.LogRepository sobre la tabla habit_logs.
type HabitLogRepository struct {
	db DBTX
}

var _ habit.LogRepository = (*HabitLogRepository)(nil)

// NewHabitLogRepository crea un HabitLogRepository que opera sobre db.
func NewHabitLogRepository(db DBTX) *HabitLogRepository {
	return &HabitLogRepository{db: db}
}

const habitLogColumns = `id, habit_id, logged_on, count, COALESCE(note, ''), created_at, updated_at`

func scanHabitLog(row pgx.Row) (habit.Log, error) {
	var l habit.Log
	err := row.Scan(uuidDst(&l.ID), uuidDst(&l.HabitID), &l.LoggedOn, &l.Count, &l.Note, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return habit.Log{}, mapError(err)
	}
	return l, nil
}

// Upsert crea el registro del día o reemplaza el existente (habit_id, logged_on)
// y completa ID, CreatedAt y UpdatedAt.
func (r *HabitLogRepository) Upsert(ctx context.Context, l *habit.Log) error {
	const q = `INSERT INTO habit_logs (habit_id, logged_on, count, note)
		VALUES ($1, $2, $3, NULLIF($4, ''))
		ON CONFLICT (habit_id, logged_on) DO UPDATE SET count = EXCLUDED.count, note = EXCLUDED.note
		RETURNING id, created_at, updated_at`
	err := r.db.QueryRow(ctx, q, uuidArg(l.HabitID), habit.DateOf(l.LoggedOn), l.Count, l.Note).
		Scan(uuidDst(&l.ID), &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return fmt.Errorf("guardar registro de hábito: %w", mapError(err))
	}
	return nil
}

// ListByHabit devuelve los registros con LoggedOn en [from, to], ordenados por fecha.
func (r *HabitLogRepository) ListByHabit(ctx context.Context, habitID domain.ID, from, to time.Time) ([]habit.Log, error) {
	q := `SELECT ` + habitLogColumns + ` FROM habit_logs
		WHERE habit_id = $1 AND logged_on BETWEEN $2 AND $3
		ORDER BY logged_on`
	rows, err := r.db.Query(ctx, q, uuidArg(habitID), habit.DateOf(from), habit.DateOf(to))
	if err != nil {
		return nil, fmt.Errorf("listar registros de hábito: %w", mapError(err))
	}
	logs, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (habit.Log, error) {
		return scanHabitLog(row)
	})
	if err != nil {
		return nil, fmt.Errorf("listar registros de hábito: %w", err)
	}
	return logs, nil
}

// Delete elimina el registro del hábito para el día de loggedOn.
func (r *HabitLogRepository) Delete(ctx context.Context, habitID domain.ID, loggedOn time.Time) error {
	const q = `DELETE FROM habit_logs WHERE habit_id = $1 AND logged_on = $2`
	tag, err := r.db.Exec(ctx, q, uuidArg(habitID), habit.DateOf(loggedOn))
	if err != nil {
		return fmt.Errorf("eliminar registro de hábito: %w", mapError(err))
	}
	if err := requireAffected(tag); err != nil {
		return fmt.Errorf("eliminar registro de hábito: %w", err)
	}
	return nil
}
