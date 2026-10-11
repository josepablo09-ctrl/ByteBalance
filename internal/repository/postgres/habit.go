package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/habit"
)

// HabitRepository implementa habit.Repository sobre la tabla habits.
type HabitRepository struct {
	db DBTX
}

var _ habit.Repository = (*HabitRepository)(nil)

// NewHabitRepository crea un HabitRepository que opera sobre db.
func NewHabitRepository(db DBTX) *HabitRepository {
	return &HabitRepository{db: db}
}

const habitColumns = `id, user_id, name, COALESCE(description, ''), frequency, target_count,
	reminder_time, is_active, created_at, updated_at, deleted_at`

func scanHabit(row pgx.Row) (habit.Habit, error) {
	var (
		h         habit.Habit
		frequency string
		reminder  pgtype.Time
	)
	err := row.Scan(uuidDst(&h.ID), uuidDst(&h.UserID), &h.Name, &h.Description, &frequency, &h.TargetCount,
		&reminder, &h.IsActive, &h.CreatedAt, &h.UpdatedAt, &h.DeletedAt)
	if err != nil {
		return habit.Habit{}, mapError(err)
	}
	h.Frequency = habit.Frequency(frequency)
	h.ReminderTime = timeOfDayFromPg(reminder)
	return h, nil
}

// timeOfDayArg convierte una hora opcional en un argumento TIME o NULL.
func timeOfDayArg(t *habit.TimeOfDay) pgtype.Time {
	if t == nil {
		return pgtype.Time{}
	}
	d := time.Duration(t.Hour)*time.Hour + time.Duration(t.Minute)*time.Minute
	return pgtype.Time{Microseconds: d.Microseconds(), Valid: true}
}

// timeOfDayFromPg convierte una columna TIME nullable en una hora opcional.
// Los segundos, si los hubiera, se descartan.
func timeOfDayFromPg(t pgtype.Time) *habit.TimeOfDay {
	if !t.Valid {
		return nil
	}
	d := time.Duration(t.Microseconds) * time.Microsecond
	return &habit.TimeOfDay{Hour: int(d / time.Hour), Minute: int(d % time.Hour / time.Minute)}
}

// Create persiste h y completa ID, CreatedAt y UpdatedAt.
func (r *HabitRepository) Create(ctx context.Context, h *habit.Habit) error {
	const q = `INSERT INTO habits (user_id, name, description, frequency, target_count, reminder_time, is_active)
		VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`
	err := r.db.QueryRow(ctx, q, uuidArg(h.UserID), h.Name, h.Description, string(h.Frequency), h.TargetCount,
		timeOfDayArg(h.ReminderTime), h.IsActive).
		Scan(uuidDst(&h.ID), &h.CreatedAt, &h.UpdatedAt)
	if err != nil {
		return fmt.Errorf("crear hábito: %w", mapError(err))
	}
	return nil
}

// GetByID devuelve el hábito no eliminado con ese ID si pertenece a userID.
func (r *HabitRepository) GetByID(ctx context.Context, userID, id domain.ID) (*habit.Habit, error) {
	q := `SELECT ` + habitColumns + ` FROM habits
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`
	h, err := scanHabit(r.db.QueryRow(ctx, q, uuidArg(id), uuidArg(userID)))
	if err != nil {
		return nil, fmt.Errorf("obtener hábito: %w", err)
	}
	return &h, nil
}

// ListByUser devuelve los hábitos del usuario en orden de creación; con
// activeOnly omite los pausados.
func (r *HabitRepository) ListByUser(ctx context.Context, userID domain.ID, activeOnly bool) ([]habit.Habit, error) {
	q := `SELECT ` + habitColumns + ` FROM habits
		WHERE user_id = $1 AND deleted_at IS NULL AND (is_active OR NOT $2::boolean)
		ORDER BY created_at, id`
	rows, err := r.db.Query(ctx, q, uuidArg(userID), activeOnly)
	if err != nil {
		return nil, fmt.Errorf("listar hábitos: %w", mapError(err))
	}
	habits, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (habit.Habit, error) {
		return scanHabit(row)
	})
	if err != nil {
		return nil, fmt.Errorf("listar hábitos: %w", err)
	}
	return habits, nil
}

// Update guarda los cambios de h y actualiza UpdatedAt.
func (r *HabitRepository) Update(ctx context.Context, h *habit.Habit) error {
	const q = `UPDATE habits
		SET name = $3, description = NULLIF($4, ''), frequency = $5, target_count = $6,
			reminder_time = $7, is_active = $8
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
		RETURNING updated_at`
	err := r.db.QueryRow(ctx, q, uuidArg(h.ID), uuidArg(h.UserID), h.Name, h.Description, string(h.Frequency),
		h.TargetCount, timeOfDayArg(h.ReminderTime), h.IsActive).
		Scan(&h.UpdatedAt)
	if err != nil {
		return fmt.Errorf("actualizar hábito: %w", mapError(err))
	}
	return nil
}

// Delete marca el hábito como eliminado.
func (r *HabitRepository) Delete(ctx context.Context, userID, id domain.ID) error {
	const q = `UPDATE habits SET deleted_at = NOW()
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`
	tag, err := r.db.Exec(ctx, q, uuidArg(id), uuidArg(userID))
	if err != nil {
		return fmt.Errorf("eliminar hábito: %w", mapError(err))
	}
	if err := requireAffected(tag); err != nil {
		return fmt.Errorf("eliminar hábito: %w", err)
	}
	return nil
}
