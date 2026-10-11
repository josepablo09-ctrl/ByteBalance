package user

import (
	"context"

	"bytebalance/internal/domain"
)

// Repository define la persistencia de usuarios. Las lecturas ignoran los
// usuarios eliminados (soft delete).
//
// Las implementaciones deben devolver domain.ErrNotFound cuando el usuario no
// existe y domain.ErrConflict ante un email o chat de Telegram duplicado.
type Repository interface {
	// Create persiste u y completa ID, CreatedAt y UpdatedAt.
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id domain.ID) (*User, error)
	// GetByEmail busca sin distinguir mayúsculas.
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByTelegramChatID(ctx context.Context, chatID int64) (*User, error)
	// Update guarda los cambios de u y actualiza UpdatedAt.
	Update(ctx context.Context, u *User) error
	// Delete marca al usuario como eliminado.
	Delete(ctx context.Context, id domain.ID) error
}
