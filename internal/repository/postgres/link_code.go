package postgres

import (
	"context"
	"fmt"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/user"
)

// LinkCodeRepository implementa user.LinkCodeRepository sobre PostgreSQL.
type LinkCodeRepository struct {
	db DBTX
}

var _ user.LinkCodeRepository = (*LinkCodeRepository)(nil)

// NewLinkCodeRepository crea un LinkCodeRepository.
func NewLinkCodeRepository(db DBTX) *LinkCodeRepository {
	return &LinkCodeRepository{db: db}
}

// Replace guarda c y borra los códigos anteriores del usuario en una sola
// sentencia, así nunca quedan dos códigos vigentes.
func (r *LinkCodeRepository) Replace(ctx context.Context, c *user.LinkCode) error {
	const q = `WITH removed AS (DELETE FROM telegram_link_codes WHERE user_id = $1)
		INSERT INTO telegram_link_codes (user_id, code_hash, expires_at) VALUES ($1, $2, $3)`
	if _, err := r.db.Exec(ctx, q, uuidArg(c.UserID), c.Hash, c.ExpiresAt); err != nil {
		return fmt.Errorf("guardar código de vinculación: %w", mapError(err))
	}
	return nil
}

// Consume marca el código como usado y devuelve su usuario. El UPDATE con
// condición es atómico: dos intentos simultáneos no pueden consumirlo dos veces.
func (r *LinkCodeRepository) Consume(ctx context.Context, hash []byte, now time.Time) (domain.ID, error) {
	const q = `UPDATE telegram_link_codes SET used_at = $2
		WHERE code_hash = $1 AND used_at IS NULL AND expires_at > $2
		RETURNING user_id`
	var id domain.ID
	if err := r.db.QueryRow(ctx, q, hash, now).Scan(uuidDst(&id)); err != nil {
		return domain.ID{}, fmt.Errorf("consumir código de vinculación: %w", mapError(err))
	}
	return id, nil
}
