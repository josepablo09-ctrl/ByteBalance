package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/user"
)

func TestUserRepository(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)
	repo := NewUserRepository(tx)

	u := createTestUser(t, tx)
	if u.ID.IsZero() || u.CreatedAt.IsZero() || u.UpdatedAt.IsZero() {
		t.Fatalf("Create no completó ID y fechas: %+v", u)
	}
	if err := u.LinkTelegram(987654321); err != nil {
		t.Fatalf("LinkTelegram: %v", err)
	}
	u.FullName = "Nombre actualizado"
	if err := repo.Update(ctx, u); err != nil {
		t.Fatalf("Update: %v", err)
	}

	lookups := []struct {
		name string
		get  func() (*user.User, error)
	}{
		{name: "por id", get: func() (*user.User, error) { return repo.GetByID(ctx, u.ID) }},
		{name: "por email sin distinguir mayúsculas", get: func() (*user.User, error) { return repo.GetByEmail(ctx, strings.ToUpper(u.Email)) }},
		{name: "por chat de telegram", get: func() (*user.User, error) { return repo.GetByTelegramChatID(ctx, 987654321) }},
	}
	for _, tt := range lookups {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.get()
			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			if got.ID != u.ID || got.Email != u.Email || got.FullName != "Nombre actualizado" ||
				got.DefaultCurrency != "USD" || got.Timezone != "UTC" ||
				got.TelegramChatID == nil || *got.TelegramChatID != 987654321 || got.DeletedAt != nil {
				t.Errorf("usuario leído = %+v, se esperaba %+v", got, u)
			}
		})
	}

	t.Run("email duplicado es conflicto", func(t *testing.T) {
		expectErrInSavepoint(t, tx, domain.ErrConflict, func(db DBTX) error {
			dup := *u
			dup.Email = strings.ToUpper(u.Email)
			dup.TelegramChatID = nil
			return NewUserRepository(db).Create(ctx, &dup)
		})
	})

	t.Run("id inexistente", func(t *testing.T) {
		if _, err := repo.GetByID(ctx, domain.NewID()); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("GetByID error = %v, se esperaba ErrNotFound", err)
		}
		missing := *u
		missing.ID = domain.NewID()
		if err := repo.Update(ctx, &missing); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("Update error = %v, se esperaba ErrNotFound", err)
		}
	})

	t.Run("delete es lógico y no repetible", func(t *testing.T) {
		if err := repo.Delete(ctx, u.ID); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if _, err := repo.GetByID(ctx, u.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("GetByID tras Delete error = %v, se esperaba ErrNotFound", err)
		}
		if err := repo.Delete(ctx, u.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("segundo Delete error = %v, se esperaba ErrNotFound", err)
		}
	})
}

func TestUserRepositorySavingsGoal(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)
	repo := NewUserRepository(tx)
	u := createTestUser(t, tx)

	if got, err := repo.GetByID(ctx, u.ID); err != nil || got.MonthlySavingsGoalCents != nil {
		t.Fatalf("un usuario nuevo no debe tener meta: %+v, %v", got, err)
	}

	goal := int64(123_456_78)
	u.MonthlySavingsGoalCents = &goal
	if err := repo.Update(ctx, u); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err := repo.GetByID(ctx, u.ID)
	if err != nil || got.MonthlySavingsGoalCents == nil || *got.MonthlySavingsGoalCents != goal {
		t.Fatalf("meta leída = %v (%v), se esperaba %d", got.MonthlySavingsGoalCents, err, goal)
	}

	u.MonthlySavingsGoalCents = nil
	if err := repo.Update(ctx, u); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got, err := repo.GetByID(ctx, u.ID); err != nil || got.MonthlySavingsGoalCents != nil {
		t.Fatalf("la meta debía quitarse: %v, %v", got.MonthlySavingsGoalCents, err)
	}

	t.Run("la base rechaza una meta no positiva", func(t *testing.T) {
		expectErrInSavepoint(t, tx, domain.ErrInvalidInput, func(db DBTX) error {
			bad := *u
			zero := int64(0)
			bad.MonthlySavingsGoalCents = &zero
			return NewUserRepository(db).Update(ctx, &bad)
		})
	})
}
