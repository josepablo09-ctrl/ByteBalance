package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/finance"
)

func TestCategoryRepository(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)
	repo := NewCategoryRepository(tx)
	owner := createTestUser(t, tx)
	other := createTestUser(t, tx)

	food, err := finance.NewCategory(owner.ID, "Comida", finance.Expense)
	if err != nil {
		t.Fatalf("NewCategory: %v", err)
	}
	if err := repo.Create(ctx, food); err != nil {
		t.Fatalf("Create: %v", err)
	}
	salary, err := finance.NewCategory(owner.ID, "Salario", finance.Income)
	if err != nil {
		t.Fatalf("NewCategory: %v", err)
	}
	if err := repo.Create(ctx, salary); err != nil {
		t.Fatalf("Create: %v", err)
	}

	t.Run("get y list del dueño", func(t *testing.T) {
		got, err := repo.GetByID(ctx, owner.ID, food.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.Name != "Comida" || got.Type != finance.Expense || got.UserID != owner.ID {
			t.Errorf("categoría leída = %+v", got)
		}
		list, err := repo.ListByUser(ctx, owner.ID)
		if err != nil {
			t.Fatalf("ListByUser: %v", err)
		}
		if len(list) != 2 || list[0].ID != food.ID || list[1].ID != salary.ID {
			t.Errorf("ListByUser = %+v, se esperaban [Comida, Salario] ordenadas por tipo", list)
		}
	})

	t.Run("otro usuario no la ve", func(t *testing.T) {
		if _, err := repo.GetByID(ctx, other.ID, food.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("GetByID error = %v, se esperaba ErrNotFound", err)
		}
		if err := repo.Delete(ctx, other.ID, food.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("Delete error = %v, se esperaba ErrNotFound", err)
		}
	})

	t.Run("nombre duplicado es conflicto", func(t *testing.T) {
		expectErrInSavepoint(t, tx, domain.ErrConflict, func(db DBTX) error {
			return NewCategoryRepository(db).Create(ctx, &finance.Category{UserID: owner.ID, Name: "COMIDA", Type: finance.Expense})
		})
	})

	t.Run("usuario inexistente", func(t *testing.T) {
		expectErrInSavepoint(t, tx, domain.ErrNotFound, func(db DBTX) error {
			return NewCategoryRepository(db).Create(ctx, &finance.Category{UserID: domain.NewID(), Name: "X", Type: finance.Expense})
		})
	})

	t.Run("update y delete", func(t *testing.T) {
		food.Name = "Supermercado"
		if err := repo.Update(ctx, food); err != nil {
			t.Fatalf("Update: %v", err)
		}
		got, err := repo.GetByID(ctx, owner.ID, food.ID)
		if err != nil || got.Name != "Supermercado" {
			t.Fatalf("GetByID tras Update = %+v, %v", got, err)
		}
		if err := repo.Delete(ctx, owner.ID, food.ID); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if _, err := repo.GetByID(ctx, owner.ID, food.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("GetByID tras Delete error = %v, se esperaba ErrNotFound", err)
		}
	})
}

func TestTransactionRepository(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)
	repo := NewTransactionRepository(tx)
	owner := createTestUser(t, tx)
	other := createTestUser(t, tx)

	food := &finance.Category{UserID: owner.ID, Name: "Comida", Type: finance.Expense}
	if err := NewCategoryRepository(tx).Create(ctx, food); err != nil {
		t.Fatalf("crear categoría: %v", err)
	}

	day := func(d int) time.Time { return time.Date(2026, 10, d, 12, 0, 0, 0, time.UTC) }
	newTx := func(typ finance.EntryType, cents int64, occurredAt time.Time, category *finance.Category) *finance.Transaction {
		t.Helper()
		txn, err := finance.NewTransaction(finance.TransactionParams{
			UserID:     owner.ID,
			Type:       typ,
			Amount:     finance.Money{Cents: cents, Currency: "USD"},
			OccurredAt: occurredAt,
			Category:   category,
		})
		if err != nil {
			t.Fatalf("NewTransaction: %v", err)
		}
		if err := repo.Create(ctx, txn); err != nil {
			t.Fatalf("Create: %v", err)
		}
		return txn
	}

	lunch := newTx(finance.Expense, 1, day(1), food)
	salary := newTx(finance.Income, finance.MaxCents, day(5), nil)
	dinner := newTx(finance.Expense, 2599, day(10), food)

	t.Run("montos exactos y campos opcionales", func(t *testing.T) {
		tests := []struct {
			name         string
			id           domain.ID
			wantCents    int64
			wantCategory bool
		}{
			{name: "un centavo con categoría", id: lunch.ID, wantCents: 1, wantCategory: true},
			{name: "monto máximo sin categoría", id: salary.ID, wantCents: finance.MaxCents},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := repo.GetByID(ctx, owner.ID, tt.id)
				if err != nil {
					t.Fatalf("GetByID: %v", err)
				}
				if got.Amount.Cents != tt.wantCents || got.Amount.Currency != "USD" {
					t.Errorf("Amount = %+v, se esperaban %d centavos USD", got.Amount, tt.wantCents)
				}
				if (got.CategoryID != nil) != tt.wantCategory {
					t.Errorf("CategoryID = %v, wantCategory %v", got.CategoryID, tt.wantCategory)
				}
				if got.Description != "" {
					t.Errorf("Description = %q, se esperaba vacía", got.Description)
				}
			})
		}
	})

	t.Run("list con filtros", func(t *testing.T) {
		tests := []struct {
			name   string
			filter finance.TransactionFilter
			want   []domain.ID
		}{
			{name: "sin filtros, más reciente primero", want: []domain.ID{dinner.ID, salary.ID, lunch.ID}},
			{name: "por tipo", filter: finance.TransactionFilter{Type: finance.Income}, want: []domain.ID{salary.ID}},
			{name: "por categoría", filter: finance.TransactionFilter{CategoryID: &food.ID}, want: []domain.ID{dinner.ID, lunch.ID}},
			{name: "rango semiabierto", filter: finance.TransactionFilter{From: day(1), To: day(10)}, want: []domain.ID{salary.ID, lunch.ID}},
			{name: "paginación", filter: finance.TransactionFilter{Limit: 1, Offset: 1}, want: []domain.ID{salary.ID}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := repo.List(ctx, owner.ID, tt.filter)
				if err != nil {
					t.Fatalf("List: %v", err)
				}
				if len(got) != len(tt.want) {
					t.Fatalf("List devolvió %d transacciones, se esperaban %d", len(got), len(tt.want))
				}
				for i := range got {
					if got[i].ID != tt.want[i] {
						t.Errorf("posición %d = %v, se esperaba %v", i, got[i].ID, tt.want[i])
					}
				}
			})
		}
	})

	t.Run("otro usuario no las ve", func(t *testing.T) {
		got, err := repo.List(ctx, other.ID, finance.TransactionFilter{})
		if err != nil || len(got) != 0 {
			t.Errorf("List de otro usuario = %d transacciones, %v", len(got), err)
		}
		if _, err := repo.GetByID(ctx, other.ID, lunch.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("GetByID error = %v, se esperaba ErrNotFound", err)
		}
	})

	t.Run("update y delete", func(t *testing.T) {
		dinner.Amount.Cents = 123456789012
		dinner.Description = "Cena"
		dinner.ClearCategory()
		if err := repo.Update(ctx, dinner); err != nil {
			t.Fatalf("Update: %v", err)
		}
		got, err := repo.GetByID(ctx, owner.ID, dinner.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.Amount.Cents != 123456789012 || got.Description != "Cena" || got.CategoryID != nil {
			t.Errorf("transacción tras Update = %+v", got)
		}
		if err := repo.Delete(ctx, owner.ID, dinner.ID); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if _, err := repo.GetByID(ctx, owner.ID, dinner.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("GetByID tras Delete error = %v, se esperaba ErrNotFound", err)
		}
	})
}

func TestTransactionRepositorySavings(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)
	repo := NewTransactionRepository(tx)
	owner := createTestUser(t, tx)

	food := &finance.Category{UserID: owner.ID, Name: "Comida", Type: finance.Expense}
	if err := NewCategoryRepository(tx).Create(ctx, food); err != nil {
		t.Fatalf("crear categoría: %v", err)
	}
	at := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	for _, typ := range []finance.EntryType{finance.Saving, finance.Withdrawal} {
		txn, err := finance.NewTransaction(finance.TransactionParams{
			UserID: owner.ID, Type: typ, Amount: finance.Money{Cents: 50_000_00, Currency: "CRC"}, OccurredAt: at,
		})
		if err != nil {
			t.Fatalf("NewTransaction(%s): %v", typ, err)
		}
		if err := repo.Create(ctx, txn); err != nil {
			t.Fatalf("Create(%s): %v", typ, err)
		}
	}

	savings, err := repo.List(ctx, owner.ID, finance.TransactionFilter{Type: finance.Saving})
	if err != nil || len(savings) != 1 || savings[0].Type != finance.Saving || savings[0].Amount.Cents != 50_000_00 {
		t.Fatalf("List(saving) = %+v, %v", savings, err)
	}

	t.Run("la base rechaza un movimiento de ahorro con categoría", func(t *testing.T) {
		expectErrInSavepoint(t, tx, domain.ErrInvalidInput, func(db DBTX) error {
			// Se salta la validación del dominio para comprobar la restricción de la base.
			bad := &finance.Transaction{
				UserID: owner.ID, Type: finance.Saving, CategoryID: &food.ID,
				Amount: finance.Money{Cents: 100, Currency: "CRC"}, OccurredAt: at,
			}
			return NewTransactionRepository(db).Create(ctx, bad)
		})
	})
}
