package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/user"
)

// UserRepository implementa user.Repository sobre la tabla users.
type UserRepository struct {
	db DBTX
}

var _ user.Repository = (*UserRepository)(nil)

// NewUserRepository crea un UserRepository que opera sobre db.
func NewUserRepository(db DBTX) *UserRepository {
	return &UserRepository{db: db}
}

const userColumns = `id, email, password_hash, full_name, default_currency, timezone,
	telegram_chat_id, created_at, updated_at, deleted_at`

func scanUser(row pgx.Row) (*user.User, error) {
	var (
		u        user.User
		currency string
	)
	err := row.Scan(uuidDst(&u.ID), &u.Email, &u.PasswordHash, &u.FullName, &currency, &u.Timezone,
		&u.TelegramChatID, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err != nil {
		return nil, mapError(err)
	}
	u.DefaultCurrency = domain.Currency(currency)
	return &u, nil
}

// Create persiste u y completa ID, CreatedAt y UpdatedAt.
func (r *UserRepository) Create(ctx context.Context, u *user.User) error {
	const q = `INSERT INTO users (email, password_hash, full_name, default_currency, timezone, telegram_chat_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`
	err := r.db.QueryRow(ctx, q, u.Email, u.PasswordHash, u.FullName, string(u.DefaultCurrency), u.Timezone, u.TelegramChatID).
		Scan(uuidDst(&u.ID), &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return fmt.Errorf("crear usuario: %w", mapError(err))
	}
	return nil
}

// GetByID devuelve el usuario activo con ese ID.
func (r *UserRepository) GetByID(ctx context.Context, id domain.ID) (*user.User, error) {
	q := `SELECT ` + userColumns + ` FROM users WHERE id = $1 AND deleted_at IS NULL`
	u, err := scanUser(r.db.QueryRow(ctx, q, uuidArg(id)))
	if err != nil {
		return nil, fmt.Errorf("obtener usuario por id: %w", err)
	}
	return u, nil
}

// GetByEmail devuelve el usuario activo con ese email, sin distinguir mayúsculas.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	q := `SELECT ` + userColumns + ` FROM users WHERE lower(email) = lower($1) AND deleted_at IS NULL`
	u, err := scanUser(r.db.QueryRow(ctx, q, email))
	if err != nil {
		return nil, fmt.Errorf("obtener usuario por email: %w", err)
	}
	return u, nil
}

// GetByTelegramChatID devuelve el usuario activo vinculado a ese chat de Telegram.
func (r *UserRepository) GetByTelegramChatID(ctx context.Context, chatID int64) (*user.User, error) {
	q := `SELECT ` + userColumns + ` FROM users WHERE telegram_chat_id = $1 AND deleted_at IS NULL`
	u, err := scanUser(r.db.QueryRow(ctx, q, chatID))
	if err != nil {
		return nil, fmt.Errorf("obtener usuario por chat de telegram: %w", err)
	}
	return u, nil
}

// Update guarda los cambios de u y actualiza UpdatedAt.
func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	const q = `UPDATE users
		SET email = $2, password_hash = $3, full_name = $4, default_currency = $5, timezone = $6, telegram_chat_id = $7
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at`
	err := r.db.QueryRow(ctx, q, uuidArg(u.ID), u.Email, u.PasswordHash, u.FullName, string(u.DefaultCurrency), u.Timezone, u.TelegramChatID).
		Scan(&u.UpdatedAt)
	if err != nil {
		return fmt.Errorf("actualizar usuario: %w", mapError(err))
	}
	return nil
}

// Delete marca al usuario como eliminado.
func (r *UserRepository) Delete(ctx context.Context, id domain.ID) error {
	const q = `UPDATE users SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	tag, err := r.db.Exec(ctx, q, uuidArg(id))
	if err != nil {
		return fmt.Errorf("eliminar usuario: %w", mapError(err))
	}
	if err := requireAffected(tag); err != nil {
		return fmt.Errorf("eliminar usuario: %w", err)
	}
	return nil
}
