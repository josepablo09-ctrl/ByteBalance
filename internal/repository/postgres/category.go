package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/finance"
)

// CategoryRepository implementa finance.CategoryRepository sobre la tabla categories.
type CategoryRepository struct {
	db DBTX
}

var _ finance.CategoryRepository = (*CategoryRepository)(nil)

// NewCategoryRepository crea un CategoryRepository que opera sobre db.
func NewCategoryRepository(db DBTX) *CategoryRepository {
	return &CategoryRepository{db: db}
}

const categoryColumns = `id, user_id, name, type, created_at, updated_at, deleted_at`

func scanCategory(row pgx.Row) (finance.Category, error) {
	var (
		c   finance.Category
		typ string
	)
	err := row.Scan(uuidDst(&c.ID), uuidDst(&c.UserID), &c.Name, &typ, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt)
	if err != nil {
		return finance.Category{}, mapError(err)
	}
	c.Type = finance.EntryType(typ)
	return c, nil
}

// Create persiste c y completa ID, CreatedAt y UpdatedAt.
func (r *CategoryRepository) Create(ctx context.Context, c *finance.Category) error {
	const q = `INSERT INTO categories (user_id, name, type)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at`
	err := r.db.QueryRow(ctx, q, uuidArg(c.UserID), c.Name, string(c.Type)).
		Scan(uuidDst(&c.ID), &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("crear categoría: %w", mapError(err))
	}
	return nil
}

// GetByID devuelve la categoría activa con ese ID si pertenece a userID.
func (r *CategoryRepository) GetByID(ctx context.Context, userID, id domain.ID) (*finance.Category, error) {
	q := `SELECT ` + categoryColumns + ` FROM categories
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`
	c, err := scanCategory(r.db.QueryRow(ctx, q, uuidArg(id), uuidArg(userID)))
	if err != nil {
		return nil, fmt.Errorf("obtener categoría: %w", err)
	}
	return &c, nil
}

// ListByUser devuelve las categorías activas del usuario ordenadas por tipo y nombre.
func (r *CategoryRepository) ListByUser(ctx context.Context, userID domain.ID) ([]finance.Category, error) {
	q := `SELECT ` + categoryColumns + ` FROM categories
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY type, lower(name)`
	rows, err := r.db.Query(ctx, q, uuidArg(userID))
	if err != nil {
		return nil, fmt.Errorf("listar categorías: %w", mapError(err))
	}
	categories, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (finance.Category, error) {
		return scanCategory(row)
	})
	if err != nil {
		return nil, fmt.Errorf("listar categorías: %w", err)
	}
	return categories, nil
}

// Update guarda los cambios de c y actualiza UpdatedAt.
func (r *CategoryRepository) Update(ctx context.Context, c *finance.Category) error {
	const q = `UPDATE categories SET name = $3, type = $4
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
		RETURNING updated_at`
	err := r.db.QueryRow(ctx, q, uuidArg(c.ID), uuidArg(c.UserID), c.Name, string(c.Type)).Scan(&c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("actualizar categoría: %w", mapError(err))
	}
	return nil
}

// Delete marca la categoría como eliminada.
func (r *CategoryRepository) Delete(ctx context.Context, userID, id domain.ID) error {
	const q = `UPDATE categories SET deleted_at = NOW()
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`
	tag, err := r.db.Exec(ctx, q, uuidArg(id), uuidArg(userID))
	if err != nil {
		return fmt.Errorf("eliminar categoría: %w", mapError(err))
	}
	if err := requireAffected(tag); err != nil {
		return fmt.Errorf("eliminar categoría: %w", err)
	}
	return nil
}
