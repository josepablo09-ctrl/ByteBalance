package finance

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"bytebalance/internal/domain"
)

const maxDescriptionLen = 255

// Transaction es un ingreso o gasto registrado por un usuario.
type Transaction struct {
	ID     domain.ID
	UserID domain.ID
	// CategoryID es nil si la transacción no está categorizada.
	CategoryID  *domain.ID
	Type        EntryType
	Amount      Money
	Description string
	OccurredAt  time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// TransactionParams agrupa los datos necesarios para crear una Transaction.
type TransactionParams struct {
	UserID      domain.ID
	Type        EntryType
	Amount      Money
	Description string
	OccurredAt  time.Time
	// Category es opcional; si se indica, debe ser del mismo usuario y tipo.
	Category *Category
}

// NewTransaction devuelve una Transaction válida sin persistir.
func NewTransaction(p TransactionParams) (*Transaction, error) {
	t := &Transaction{
		UserID:      p.UserID,
		Type:        p.Type,
		Amount:      p.Amount,
		Description: strings.TrimSpace(p.Description),
		OccurredAt:  p.OccurredAt,
	}
	if err := t.Validate(); err != nil {
		return nil, err
	}
	if p.Category != nil {
		if err := t.AssignCategory(p.Category); err != nil {
			return nil, err
		}
	}
	return t, nil
}

// Validate comprueba las invariantes de la transacción.
func (t *Transaction) Validate() error {
	if t.UserID.IsZero() {
		return fmt.Errorf("%w: la transacción debe pertenecer a un usuario", domain.ErrInvalidInput)
	}
	if !t.Type.Valid() {
		return fmt.Errorf("%w: tipo de transacción %q inválido", domain.ErrInvalidInput, t.Type)
	}
	if err := t.Amount.Validate(); err != nil {
		return err
	}
	if utf8.RuneCountInString(t.Description) > maxDescriptionLen {
		return fmt.Errorf("%w: la descripción admite hasta %d caracteres", domain.ErrInvalidInput, maxDescriptionLen)
	}
	if t.OccurredAt.IsZero() {
		return fmt.Errorf("%w: falta la fecha de la transacción", domain.ErrInvalidInput)
	}
	return nil
}

// AssignCategory categoriza la transacción. La categoría debe estar activa,
// pertenecer al mismo usuario y ser del mismo tipo (ingreso/gasto).
func (t *Transaction) AssignCategory(c *Category) error {
	switch {
	case c.ID.IsZero():
		return fmt.Errorf("%w: la categoría no está persistida", domain.ErrInvalidInput)
	case c.DeletedAt != nil:
		return fmt.Errorf("%w: la categoría fue eliminada", domain.ErrInvalidInput)
	case c.UserID != t.UserID:
		// Se reporta como inexistente para no revelar categorías de otros usuarios.
		return fmt.Errorf("categoría de la transacción: %w", domain.ErrNotFound)
	case c.Type != t.Type:
		return fmt.Errorf("%w: una categoría de tipo %q no aplica a una transacción de tipo %q", domain.ErrInvalidInput, c.Type, t.Type)
	}
	id := c.ID
	t.CategoryID = &id
	return nil
}

// ClearCategory deja la transacción sin categoría.
func (t *Transaction) ClearCategory() {
	t.CategoryID = nil
}

// SignedCents devuelve el monto con signo: positivo para ingresos y negativo
// para gastos, útil para calcular balances.
func (t *Transaction) SignedCents() int64 {
	if t.Type == Expense {
		return -t.Amount.Cents
	}
	return t.Amount.Cents
}
