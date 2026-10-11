package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/finance"
)

// TransactionRepository implementa finance.TransactionRepository sobre la tabla transactions.
//
// Los montos viajan como centavos (BIGINT) y la conversión a NUMERIC(14,2) se
// hace en SQL, de modo que nunca pasan por punto flotante.
type TransactionRepository struct {
	db DBTX
}

var _ finance.TransactionRepository = (*TransactionRepository)(nil)

// NewTransactionRepository crea un TransactionRepository que opera sobre db.
func NewTransactionRepository(db DBTX) *TransactionRepository {
	return &TransactionRepository{db: db}
}

const transactionColumns = `id, user_id, category_id, type, (amount * 100)::bigint, currency,
	COALESCE(description, ''), occurred_at, created_at, updated_at, deleted_at`

func scanTransaction(row pgx.Row) (finance.Transaction, error) {
	var (
		t          finance.Transaction
		categoryID pgtype.UUID
		typ        string
		currency   string
	)
	err := row.Scan(uuidDst(&t.ID), uuidDst(&t.UserID), &categoryID, &typ, &t.Amount.Cents, &currency,
		&t.Description, &t.OccurredAt, &t.CreatedAt, &t.UpdatedAt, &t.DeletedAt)
	if err != nil {
		return finance.Transaction{}, mapError(err)
	}
	t.CategoryID = idFromNullable(categoryID)
	t.Type = finance.EntryType(typ)
	t.Amount.Currency = domain.Currency(currency)
	return t, nil
}

// Create persiste t y completa ID, CreatedAt y UpdatedAt.
func (r *TransactionRepository) Create(ctx context.Context, t *finance.Transaction) error {
	const q = `INSERT INTO transactions (user_id, category_id, type, amount, currency, description, occurred_at)
		VALUES ($1, $2, $3, $4::numeric / 100, $5, NULLIF($6, ''), $7)
		RETURNING id, created_at, updated_at`
	err := r.db.QueryRow(ctx, q, uuidArg(t.UserID), nullableUUIDArg(t.CategoryID), string(t.Type),
		t.Amount.Cents, string(t.Amount.Currency), t.Description, t.OccurredAt).
		Scan(uuidDst(&t.ID), &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return fmt.Errorf("crear transacción: %w", mapError(err))
	}
	return nil
}

// GetByID devuelve la transacción activa con ese ID si pertenece a userID.
func (r *TransactionRepository) GetByID(ctx context.Context, userID, id domain.ID) (*finance.Transaction, error) {
	q := `SELECT ` + transactionColumns + ` FROM transactions
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`
	t, err := scanTransaction(r.db.QueryRow(ctx, q, uuidArg(id), uuidArg(userID)))
	if err != nil {
		return nil, fmt.Errorf("obtener transacción: %w", err)
	}
	return &t, nil
}

// List devuelve las transacciones activas del usuario que cumplen f, ordenadas
// por OccurredAt descendente.
func (r *TransactionRepository) List(ctx context.Context, userID domain.ID, f finance.TransactionFilter) ([]finance.Transaction, error) {
	// Solo se concatenan fragmentos fijos; todos los valores van como parámetros.
	q := `SELECT ` + transactionColumns + ` FROM transactions WHERE user_id = $1 AND deleted_at IS NULL`
	args := []any{uuidArg(userID)}
	addArg := func(clause string, v any) {
		args = append(args, v)
		q += fmt.Sprintf(clause, len(args))
	}
	if !f.From.IsZero() {
		addArg(" AND occurred_at >= $%d", f.From)
	}
	if !f.To.IsZero() {
		addArg(" AND occurred_at < $%d", f.To)
	}
	if f.Type != "" {
		addArg(" AND type = $%d", string(f.Type))
	}
	if f.CategoryID != nil {
		addArg(" AND category_id = $%d", uuidArg(*f.CategoryID))
	}
	q += " ORDER BY occurred_at DESC, id DESC"
	if f.Limit > 0 {
		addArg(" LIMIT $%d", f.Limit)
	}
	if f.Offset > 0 {
		addArg(" OFFSET $%d", f.Offset)
	}

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("listar transacciones: %w", mapError(err))
	}
	transactions, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (finance.Transaction, error) {
		return scanTransaction(row)
	})
	if err != nil {
		return nil, fmt.Errorf("listar transacciones: %w", err)
	}
	return transactions, nil
}

// Update guarda los cambios de t y actualiza UpdatedAt.
func (r *TransactionRepository) Update(ctx context.Context, t *finance.Transaction) error {
	const q = `UPDATE transactions
		SET category_id = $3, type = $4, amount = $5::numeric / 100, currency = $6,
			description = NULLIF($7, ''), occurred_at = $8
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
		RETURNING updated_at`
	err := r.db.QueryRow(ctx, q, uuidArg(t.ID), uuidArg(t.UserID), nullableUUIDArg(t.CategoryID), string(t.Type),
		t.Amount.Cents, string(t.Amount.Currency), t.Description, t.OccurredAt).
		Scan(&t.UpdatedAt)
	if err != nil {
		return fmt.Errorf("actualizar transacción: %w", mapError(err))
	}
	return nil
}

// Delete marca la transacción como eliminada.
func (r *TransactionRepository) Delete(ctx context.Context, userID, id domain.ID) error {
	const q = `UPDATE transactions SET deleted_at = NOW()
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`
	tag, err := r.db.Exec(ctx, q, uuidArg(id), uuidArg(userID))
	if err != nil {
		return fmt.Errorf("eliminar transacción: %w", mapError(err))
	}
	if err := requireAffected(tag); err != nil {
		return fmt.Errorf("eliminar transacción: %w", err)
	}
	return nil
}
