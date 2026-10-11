package finance

import (
	"context"
	"time"

	"bytebalance/internal/domain"
)

// CategoryRepository define la persistencia de categorías. Todas las operaciones
// se acotan al usuario dueño; las lecturas ignoran las categorías eliminadas.
//
// Las implementaciones deben devolver domain.ErrNotFound si la categoría no
// existe para ese usuario y domain.ErrConflict ante un nombre duplicado del mismo tipo.
type CategoryRepository interface {
	// Create persiste c y completa ID, CreatedAt y UpdatedAt.
	Create(ctx context.Context, c *Category) error
	GetByID(ctx context.Context, userID, id domain.ID) (*Category, error)
	ListByUser(ctx context.Context, userID domain.ID) ([]Category, error)
	// Update guarda los cambios de c y actualiza UpdatedAt.
	Update(ctx context.Context, c *Category) error
	// Delete marca la categoría como eliminada.
	Delete(ctx context.Context, userID, id domain.ID) error
}

// TransactionFilter acota un listado de transacciones. Los campos en su valor
// cero no filtran.
type TransactionFilter struct {
	// From y To delimitan OccurredAt en el intervalo [From, To).
	From       time.Time
	To         time.Time
	Type       EntryType
	CategoryID *domain.ID
	Limit      int
	Offset     int
}

// TransactionRepository define la persistencia de transacciones. Todas las
// operaciones se acotan al usuario dueño; las lecturas ignoran las eliminadas.
//
// Las implementaciones deben devolver domain.ErrNotFound si la transacción no
// existe para ese usuario.
type TransactionRepository interface {
	// Create persiste t y completa ID, CreatedAt y UpdatedAt.
	Create(ctx context.Context, t *Transaction) error
	GetByID(ctx context.Context, userID, id domain.ID) (*Transaction, error)
	// List devuelve las transacciones ordenadas por OccurredAt descendente.
	List(ctx context.Context, userID domain.ID, f TransactionFilter) ([]Transaction, error)
	// Update guarda los cambios de t y actualiza UpdatedAt.
	Update(ctx context.Context, t *Transaction) error
	// Delete marca la transacción como eliminada.
	Delete(ctx context.Context, userID, id domain.ID) error
}
