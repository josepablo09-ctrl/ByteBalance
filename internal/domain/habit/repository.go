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

// Reminder es un recordatorio a enviar: un hábito activo con recordatorio cuyo
// dueño tiene Telegram vinculado.
type Reminder struct {
	Habit Habit
	// Timezone es la zona horaria IANA del dueño, en la que se interpreta la hora.
	Timezone string
	ChatID   int64
}

// ReminderRepository busca los recordatorios que tocan en un momento dado.
type ReminderRepository interface {
	// ListDue devuelve los recordatorios cuya hora coincide con el minuto de at
	// en la zona horaria de cada dueño. Excluye hábitos pausados o eliminados y
	// usuarios sin Telegram vinculado.
	ListDue(ctx context.Context, at time.Time) ([]Reminder, error)
}
