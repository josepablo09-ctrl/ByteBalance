package habit

import (
	"context"
	"time"

	"bytebalance/internal/domain"
)

// Repository define la persistencia de hábitos. Todas las operaciones se acotan
// al usuario dueño; las lecturas ignoran los hábitos eliminados.
//
// Las implementaciones deben devolver domain.ErrNotFound si el hábito no existe
// para ese usuario.
type Repository interface {
	// Create persiste h y completa ID, CreatedAt y UpdatedAt.
	Create(ctx context.Context, h *Habit) error
	GetByID(ctx context.Context, userID, id domain.ID) (*Habit, error)
	// ListByUser devuelve los hábitos del usuario; con activeOnly omite los pausados.
	ListByUser(ctx context.Context, userID domain.ID, activeOnly bool) ([]Habit, error)
	// Update guarda los cambios de h y actualiza UpdatedAt.
	Update(ctx context.Context, h *Habit) error
	// Delete marca el hábito como eliminado.
	Delete(ctx context.Context, userID, id domain.ID) error
}

// LogRepository define la persistencia de los registros de hábitos. No valida
// la propiedad del hábito: el caso de uso debe comprobarla antes con Repository.
//
// Las implementaciones deben devolver domain.ErrNotFound si el registro no existe.
type LogRepository interface {
	// Upsert crea el registro del día o reemplaza el existente (habit_id, logged_on)
	// y completa ID, CreatedAt y UpdatedAt.
	Upsert(ctx context.Context, l *Log) error
	// ListByHabit devuelve los registros con LoggedOn en [from, to], ordenados por fecha.
	ListByHabit(ctx context.Context, habitID domain.ID, from, to time.Time) ([]Log, error)
	Delete(ctx context.Context, habitID domain.ID, loggedOn time.Time) error
}
