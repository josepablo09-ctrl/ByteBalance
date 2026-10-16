package postgres

import (
	"context"
	"testing"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/user"
)

func TestLinkCodeRepository(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)
	repo := NewLinkCodeRepository(tx)
	owner := createTestUser(t, tx)
	now := time.Now().UTC()

	newCode := func(t *testing.T) (string, *user.LinkCode) {
		t.Helper()
		code, lc, err := user.NewLinkCode(owner.ID, now)
		if err != nil {
			t.Fatalf("NewLinkCode: %v", err)
		}
		if err := repo.Replace(ctx, lc); err != nil {
			t.Fatalf("Replace: %v", err)
		}
		return code, lc
	}

	t.Run("se consume una sola vez", func(t *testing.T) {
		code, _ := newCode(t)
		got, err := repo.Consume(ctx, user.HashLinkCode(code), now)
		if err != nil || got != owner.ID {
			t.Fatalf("Consume() = %v, %v; se esperaba el usuario %v", got, err, owner.ID)
		}
		expectErrInSavepoint(t, tx, domain.ErrNotFound, func(db DBTX) error {
			_, err := NewLinkCodeRepository(db).Consume(ctx, user.HashLinkCode(code), now)
			return err
		})
	})

	t.Run("un código nuevo invalida el anterior", func(t *testing.T) {
		first, _ := newCode(t)
		second, _ := newCode(t)
		expectErrInSavepoint(t, tx, domain.ErrNotFound, func(db DBTX) error {
			_, err := NewLinkCodeRepository(db).Consume(ctx, user.HashLinkCode(first), now)
			return err
		})
		if _, err := repo.Consume(ctx, user.HashLinkCode(second), now); err != nil {
			t.Errorf("el último código debía seguir vigente: %v", err)
		}
	})

	t.Run("vencido", func(t *testing.T) {
		code, lc := newCode(t)
		expectErrInSavepoint(t, tx, domain.ErrNotFound, func(db DBTX) error {
			_, err := NewLinkCodeRepository(db).Consume(ctx, user.HashLinkCode(code), lc.ExpiresAt)
			return err
		})
	})

	t.Run("inexistente", func(t *testing.T) {
		expectErrInSavepoint(t, tx, domain.ErrNotFound, func(db DBTX) error {
			_, err := NewLinkCodeRepository(db).Consume(ctx, user.HashLinkCode("no-existe"), now)
			return err
		})
	})

	t.Run("hash con largo inválido", func(t *testing.T) {
		expectErrInSavepoint(t, tx, domain.ErrInvalidInput, func(db DBTX) error {
			return NewLinkCodeRepository(db).Replace(ctx, &user.LinkCode{UserID: owner.ID, Hash: []byte("corto"), ExpiresAt: now})
		})
	})
}
