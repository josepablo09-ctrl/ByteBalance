package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/finance"
)

// savingsFixture tiene un usuario con 100 000 CRC apartados y 30 000 retirados
// (ahorro actual: 70 000 CRC), además de ingresos y gastos.
type savingsFixture struct {
	owner        domain.ID
	food         finance.Category
	transactions *fakeTransactionRepo
	svc          *FinanceService
	deposit      finance.Transaction
	withdrawal   finance.Transaction
	expense      finance.Transaction
}

func newSavingsFixture(t *testing.T) *savingsFixture {
	t.Helper()
	u := existingUser(t, "ana@example.com", "UTC")
	f := &savingsFixture{owner: u.ID, transactions: &fakeTransactionRepo{}}
	f.food = finance.Category{ID: domain.NewID(), UserID: u.ID, Name: "Comida", Type: finance.Expense}
	tx := func(typ finance.EntryType, cents int64, category *domain.ID) finance.Transaction {
		return finance.Transaction{
			ID: domain.NewID(), UserID: u.ID, Type: typ, CategoryID: category,
			Amount: finance.Money{Cents: cents, Currency: "CRC"}, OccurredAt: fixedNow,
		}
	}
	f.deposit = tx(finance.Saving, 100_000_00, nil)
	f.withdrawal = tx(finance.Withdrawal, 30_000_00, nil)
	f.expense = tx(finance.Expense, 50_000_00, &f.food.ID)
	f.transactions.stored = []finance.Transaction{tx(finance.Income, 500_000_00, nil), f.expense, f.deposit, f.withdrawal}
	f.svc = NewFinanceService(newFakeCategoryRepo(f.food), f.transactions, newFakeUserRepo(u), fixedClock)
	return f
}

func TestFinanceServiceRecordSavings(t *testing.T) {
	ctx := context.Background()
	foodID := func(f *savingsFixture) *domain.ID { return &f.food.ID }

	tests := []struct {
		name       string
		in         RecordTransactionInput
		categoryID func(f *savingsFixture) *domain.ID
		wantErr    error
		wantMsg    string
	}{
		{name: "aporte", in: RecordTransactionInput{Type: finance.Saving, AmountCents: 1_000_000_00}},
		{name: "retiro dentro del ahorro", in: RecordTransactionInput{Type: finance.Withdrawal, AmountCents: 70_000_00}},
		{
			name:    "retiro mayor que el ahorro",
			in:      RecordTransactionInput{Type: finance.Withdrawal, AmountCents: 70_000_01},
			wantErr: domain.ErrInvalidInput, wantMsg: "ahorro en CRC: 70000.00",
		},
		{
			name:    "retiro en una moneda sin ahorro",
			in:      RecordTransactionInput{Type: finance.Withdrawal, AmountCents: 1, Currency: "USD"},
			wantErr: domain.ErrInvalidInput, wantMsg: "ahorro en USD: 0.00",
		},
		{
			name:       "ahorro con categoría",
			in:         RecordTransactionInput{Type: finance.Saving, AmountCents: 100},
			categoryID: foodID,
			wantErr:    domain.ErrInvalidInput,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newSavingsFixture(t)
			in := tt.in
			if tt.categoryID != nil {
				in.CategoryID = tt.categoryID(f)
			}
			before := len(f.transactions.stored)

			got, err := f.svc.RecordTransaction(ctx, f.owner, in)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) || !strings.Contains(err.Error(), tt.wantMsg) {
					t.Fatalf("RecordTransaction() error = %v, se esperaba %v con %q", err, tt.wantErr, tt.wantMsg)
				}
				if len(f.transactions.stored) != before {
					t.Error("un movimiento rechazado no debe guardarse")
				}
				return
			}
			if err != nil {
				t.Fatalf("RecordTransaction() error inesperado: %v", err)
			}
			if got.Type != in.Type || got.CategoryID != nil || got.Amount.Currency != "CRC" {
				t.Errorf("transacción = %+v", got)
			}
		})
	}
}

func TestFinanceServiceUpdateSavings(t *testing.T) {
	ctx := context.Background()
	typ := func(t finance.EntryType) *finance.EntryType { return &t }
	cents := func(c int64) *int64 { return &c }

	tests := []struct {
		name    string
		target  func(f *savingsFixture) domain.ID
		in      UpdateTransactionInput
		wantErr error
		check   func(t *testing.T, got *finance.Transaction)
	}{
		{
			name:   "aumentar el retiro hasta el ahorro disponible",
			target: func(f *savingsFixture) domain.ID { return f.withdrawal.ID },
			in:     UpdateTransactionInput{AmountCents: cents(100_000_00)},
		},
		{
			name:    "aumentar el retiro por encima del ahorro",
			target:  func(f *savingsFixture) domain.ID { return f.withdrawal.ID },
			in:      UpdateTransactionInput{AmountCents: cents(100_000_01)},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:    "reducir el aporte por debajo de lo retirado",
			target:  func(f *savingsFixture) domain.ID { return f.deposit.ID },
			in:      UpdateTransactionInput{AmountCents: cents(29_999_99)},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:    "convertir el aporte en gasto deja el ahorro en negativo",
			target:  func(f *savingsFixture) domain.ID { return f.deposit.ID },
			in:      UpdateTransactionInput{Type: typ(finance.Expense)},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:   "convertir un gasto en ahorro le quita la categoría",
			target: func(f *savingsFixture) domain.ID { return f.expense.ID },
			in:     UpdateTransactionInput{Type: typ(finance.Saving)},
			check: func(t *testing.T, got *finance.Transaction) {
				if got.Type != finance.Saving || got.CategoryID != nil {
					t.Errorf("transacción = %+v, se esperaba ahorro sin categoría", got)
				}
			},
		},
		{
			name:    "convertir un gasto en un retiro mayor que el ahorro",
			target:  func(f *savingsFixture) domain.ID { return f.expense.ID },
			in:      UpdateTransactionInput{Type: typ(finance.Withdrawal), AmountCents: cents(70_000_01)},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:   "cambiar la moneda del retiro a una sin ahorro",
			target: func(f *savingsFixture) domain.ID { return f.withdrawal.ID },
			in: UpdateTransactionInput{Currency: func() *string {
				s := "USD"
				return &s
			}()},
			wantErr: domain.ErrInvalidInput,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newSavingsFixture(t)
			snapshot := append([]finance.Transaction(nil), f.transactions.stored...)

			got, err := f.svc.UpdateTransaction(ctx, f.owner, tt.target(f), tt.in)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("UpdateTransaction() error = %v, se esperaba %v", err, tt.wantErr)
				}
				for i := range snapshot {
					if !sameTransaction(snapshot[i], f.transactions.stored[i]) {
						t.Fatalf("un cambio rechazado no debe guardarse: %+v", f.transactions.stored[i])
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("UpdateTransaction() error inesperado: %v", err)
			}
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestFinanceServiceDeleteSavings(t *testing.T) {
	ctx := context.Background()
	t.Run("no se puede borrar un aporte ya retirado en parte", func(t *testing.T) {
		f := newSavingsFixture(t)
		err := f.svc.DeleteTransaction(ctx, f.owner, f.deposit.ID)
		if !errors.Is(err, domain.ErrInvalidInput) || !strings.Contains(err.Error(), "retiros") {
			t.Fatalf("DeleteTransaction() error = %v, se esperaba ErrInvalidInput", err)
		}
		if len(f.transactions.stored) != 4 {
			t.Error("el aporte no debía borrarse")
		}
	})
	t.Run("borrar el retiro y luego el aporte", func(t *testing.T) {
		f := newSavingsFixture(t)
		if err := f.svc.DeleteTransaction(ctx, f.owner, f.withdrawal.ID); err != nil {
			t.Fatalf("borrar retiro: %v", err)
		}
		if err := f.svc.DeleteTransaction(ctx, f.owner, f.deposit.ID); err != nil {
			t.Fatalf("borrar aporte: %v", err)
		}
	})
	t.Run("borrar un gasto no consulta el ahorro", func(t *testing.T) {
		f := newSavingsFixture(t)
		f.transactions.listErr = errors.New("no debe listarse")
		if err := f.svc.DeleteTransaction(ctx, f.owner, f.expense.ID); err != nil {
			t.Fatalf("DeleteTransaction() error inesperado: %v", err)
		}
	})
}

func TestFinanceServiceBalanceWithSavings(t *testing.T) {
	f := newSavingsFixture(t)
	balances, err := f.svc.Balance(context.Background(), f.owner, fixedNow.AddDate(0, 0, -1), fixedNow.AddDate(0, 0, 1))
	if err != nil {
		t.Fatalf("Balance() error inesperado: %v", err)
	}
	if len(balances) != 1 {
		t.Fatalf("Balance() = %+v", balances)
	}
	b := balances[0]
	// 500 000 - 50 000 = 450 000 de balance; 70 000 apartados → 380 000 disponibles.
	if b.NetCents() != 450_000_00 || b.SavingsCents() != 70_000_00 || b.AvailableCents() != 380_000_00 {
		t.Errorf("balance = %+v (neto %d, ahorro %d, disponible %d)", b, b.NetCents(), b.SavingsCents(), b.AvailableCents())
	}
}

func sameTransaction(a, b finance.Transaction) bool {
	sameCategory := (a.CategoryID == nil) == (b.CategoryID == nil) && (a.CategoryID == nil || *a.CategoryID == *b.CategoryID)
	return a.ID == b.ID && a.Type == b.Type && a.Amount == b.Amount && sameCategory
}
