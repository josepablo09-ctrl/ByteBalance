package usecase

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/finance"
)

// fakeCategoryRepo es un finance.CategoryRepository en memoria acotado por usuario.
type fakeCategoryRepo struct {
	byID map[domain.ID]finance.Category
}

var _ finance.CategoryRepository = (*fakeCategoryRepo)(nil)

func newFakeCategoryRepo(categories ...finance.Category) *fakeCategoryRepo {
	f := &fakeCategoryRepo{byID: map[domain.ID]finance.Category{}}
	for _, c := range categories {
		f.byID[c.ID] = c
	}
	return f
}

func (f *fakeCategoryRepo) Create(_ context.Context, c *finance.Category) error {
	for _, existing := range f.byID {
		if existing.UserID == c.UserID && existing.Type == c.Type && strings.EqualFold(existing.Name, c.Name) {
			return domain.ErrConflict
		}
	}
	c.ID = domain.NewID()
	f.byID[c.ID] = *c
	return nil
}

func (f *fakeCategoryRepo) GetByID(_ context.Context, userID, id domain.ID) (*finance.Category, error) {
	c, ok := f.byID[id]
	if !ok || c.UserID != userID {
		return nil, domain.ErrNotFound
	}
	return &c, nil
}

func (f *fakeCategoryRepo) ListByUser(_ context.Context, userID domain.ID) ([]finance.Category, error) {
	var categories []finance.Category
	for _, c := range f.byID {
		if c.UserID == userID {
			categories = append(categories, c)
		}
	}
	slices.SortFunc(categories, func(a, b finance.Category) int { return cmp.Compare(a.Name, b.Name) })
	return categories, nil
}

func (f *fakeCategoryRepo) Update(_ context.Context, c *finance.Category) error {
	if _, err := f.GetByID(context.Background(), c.UserID, c.ID); err != nil {
		return err
	}
	f.byID[c.ID] = *c
	return nil
}

func (f *fakeCategoryRepo) Delete(_ context.Context, userID, id domain.ID) error {
	if _, err := f.GetByID(context.Background(), userID, id); err != nil {
		return err
	}
	delete(f.byID, id)
	return nil
}

// fakeTransactionRepo es un finance.TransactionRepository en memoria. Registra
// el último filtro recibido para verificar lo que el caso de uso envía.
type fakeTransactionRepo struct {
	stored     []finance.Transaction
	listErr    error
	lastFilter finance.TransactionFilter
}

var _ finance.TransactionRepository = (*fakeTransactionRepo)(nil)

func (f *fakeTransactionRepo) Create(_ context.Context, t *finance.Transaction) error {
	t.ID = domain.NewID()
	f.stored = append(f.stored, *t)
	return nil
}

func (f *fakeTransactionRepo) GetByID(_ context.Context, userID, id domain.ID) (*finance.Transaction, error) {
	for _, t := range f.stored {
		if t.ID == id && t.UserID == userID {
			return &t, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeTransactionRepo) List(_ context.Context, userID domain.ID, filter finance.TransactionFilter) ([]finance.Transaction, error) {
	f.lastFilter = filter
	if f.listErr != nil {
		return nil, f.listErr
	}
	var result []finance.Transaction
	for _, t := range f.stored {
		switch {
		case t.UserID != userID,
			!filter.From.IsZero() && t.OccurredAt.Before(filter.From),
			!filter.To.IsZero() && !t.OccurredAt.Before(filter.To),
			filter.Type != "" && t.Type != filter.Type:
			continue
		}
		result = append(result, t)
	}
	return result, nil
}

func (f *fakeTransactionRepo) Update(_ context.Context, t *finance.Transaction) error {
	for i, stored := range f.stored {
		if stored.ID == t.ID && stored.UserID == t.UserID {
			f.stored[i] = *t
			return nil
		}
	}
	return domain.ErrNotFound
}

func (f *fakeTransactionRepo) Delete(_ context.Context, userID, id domain.ID) error {
	for i, stored := range f.stored {
		if stored.ID == id && stored.UserID == userID {
			f.stored = slices.Delete(f.stored, i, i+1)
			return nil
		}
	}
	return domain.ErrNotFound
}

var fixedNow = time.Date(2026, 10, 11, 18, 30, 0, 0, time.UTC)

func fixedClock() time.Time { return fixedNow }

func TestFinanceServiceCreateCategory(t *testing.T) {
	owner := domain.NewID()
	tests := []struct {
		name     string
		catName  string
		catType  finance.EntryType
		wantErr  error
		wantSize int
	}{
		{name: "válida", catName: " Transporte ", catType: finance.Expense, wantSize: 2},
		{name: "mismo nombre en otro tipo", catName: "Comida", catType: finance.Income, wantSize: 2},
		{name: "duplicada", catName: "comida", catType: finance.Expense, wantErr: domain.ErrConflict, wantSize: 1},
		{name: "nombre vacío", catName: " ", catType: finance.Expense, wantErr: domain.ErrInvalidInput, wantSize: 1},
		{name: "tipo inválido", catName: "Ahorro", catType: "saving", wantErr: domain.ErrInvalidInput, wantSize: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			categories := newFakeCategoryRepo(finance.Category{ID: domain.NewID(), UserID: owner, Name: "Comida", Type: finance.Expense})
			svc := NewFinanceService(categories, &fakeTransactionRepo{}, newFakeUserRepo(), fixedClock)

			c, err := svc.CreateCategory(context.Background(), owner, tt.catName, tt.catType)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("CreateCategory() error = %v, se esperaba %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("CreateCategory() error inesperado: %v", err)
			} else if c.ID.IsZero() || c.Name != strings.TrimSpace(tt.catName) {
				t.Errorf("categoría creada = %+v", c)
			}
			if len(categories.byID) != tt.wantSize {
				t.Errorf("hay %d categorías, se esperaban %d", len(categories.byID), tt.wantSize)
			}
		})
	}
}

func TestFinanceServiceListCategories(t *testing.T) {
	owner, other := domain.NewID(), domain.NewID()
	categories := newFakeCategoryRepo(
		finance.Category{ID: domain.NewID(), UserID: owner, Name: "Comida", Type: finance.Expense},
		finance.Category{ID: domain.NewID(), UserID: other, Name: "Ajena", Type: finance.Expense},
	)
	svc := NewFinanceService(categories, &fakeTransactionRepo{}, newFakeUserRepo(), fixedClock)

	got, err := svc.ListCategories(context.Background(), owner)
	if err != nil {
		t.Fatalf("ListCategories() error inesperado: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Comida" {
		t.Errorf("ListCategories() = %+v, se esperaba solo la categoría del usuario", got)
	}
}

func TestFinanceServiceRecordTransaction(t *testing.T) {
	owner := existingUser(t, "ana@example.com", "UTC") // moneda por defecto CRC
	stranger := domain.NewID()
	food := finance.Category{ID: domain.NewID(), UserID: owner.ID, Name: "Comida", Type: finance.Expense}
	salary := finance.Category{ID: domain.NewID(), UserID: owner.ID, Name: "Salario", Type: finance.Income}
	foreign := finance.Category{ID: domain.NewID(), UserID: stranger, Name: "Ajena", Type: finance.Expense}
	missingID := domain.NewID()
	yesterday := fixedNow.Add(-24 * time.Hour)

	tests := []struct {
		name    string
		userID  domain.ID
		in      RecordTransactionInput
		wantErr error
		check   func(t *testing.T, tx *finance.Transaction)
	}{
		{
			name:   "gasto con categoría y valores por defecto",
			userID: owner.ID,
			in:     RecordTransactionInput{Type: finance.Expense, AmountCents: 2599, CategoryID: &food.ID, Description: " Almuerzo "},
			check: func(t *testing.T, tx *finance.Transaction) {
				if tx.Amount != (finance.Money{Cents: 2599, Currency: "CRC"}) {
					t.Errorf("Amount = %+v, se esperaban 2599 centavos en la moneda del usuario", tx.Amount)
				}
				if tx.CategoryID == nil || *tx.CategoryID != food.ID || tx.Description != "Almuerzo" {
					t.Errorf("transacción = %+v", tx)
				}
				if !tx.OccurredAt.Equal(fixedNow) {
					t.Errorf("OccurredAt = %v, se esperaba el momento actual %v", tx.OccurredAt, fixedNow)
				}
			},
		},
		{
			name:   "ingreso con moneda y fecha explícitas",
			userID: owner.ID,
			in:     RecordTransactionInput{Type: finance.Income, AmountCents: 150000, Currency: "usd", CategoryID: &salary.ID, OccurredAt: yesterday},
			check: func(t *testing.T, tx *finance.Transaction) {
				if tx.Amount.Currency != "USD" || !tx.OccurredAt.Equal(yesterday) {
					t.Errorf("transacción = %+v", tx)
				}
			},
		},
		{
			name:   "sin categoría",
			userID: owner.ID,
			in:     RecordTransactionInput{Type: finance.Expense, AmountCents: 1},
			check: func(t *testing.T, tx *finance.Transaction) {
				if tx.CategoryID != nil {
					t.Errorf("CategoryID = %v, se esperaba nil", tx.CategoryID)
				}
			},
		},
		{name: "categoría de otro usuario", userID: owner.ID, in: RecordTransactionInput{Type: finance.Expense, AmountCents: 100, CategoryID: &foreign.ID}, wantErr: domain.ErrNotFound},
		{name: "categoría inexistente", userID: owner.ID, in: RecordTransactionInput{Type: finance.Expense, AmountCents: 100, CategoryID: &missingID}, wantErr: domain.ErrNotFound},
		{name: "categoría de otro tipo", userID: owner.ID, in: RecordTransactionInput{Type: finance.Expense, AmountCents: 100, CategoryID: &salary.ID}, wantErr: domain.ErrInvalidInput},
		{name: "monto cero", userID: owner.ID, in: RecordTransactionInput{Type: finance.Expense, AmountCents: 0}, wantErr: domain.ErrInvalidInput},
		{name: "monto negativo", userID: owner.ID, in: RecordTransactionInput{Type: finance.Expense, AmountCents: -500}, wantErr: domain.ErrInvalidInput},
		{name: "monto excede NUMERIC(14,2)", userID: owner.ID, in: RecordTransactionInput{Type: finance.Income, AmountCents: finance.MaxCents + 1}, wantErr: domain.ErrInvalidInput},
		{name: "moneda inválida", userID: owner.ID, in: RecordTransactionInput{Type: finance.Expense, AmountCents: 100, Currency: "US"}, wantErr: domain.ErrInvalidInput},
		{name: "tipo inválido", userID: owner.ID, in: RecordTransactionInput{Type: "transfer", AmountCents: 100}, wantErr: domain.ErrInvalidInput},
		{name: "usuario inexistente sin moneda", userID: stranger, in: RecordTransactionInput{Type: finance.Expense, AmountCents: 100}, wantErr: domain.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transactions := &fakeTransactionRepo{}
			svc := NewFinanceService(newFakeCategoryRepo(food, salary, foreign), transactions, newFakeUserRepo(owner), fixedClock)

			tx, err := svc.RecordTransaction(context.Background(), tt.userID, tt.in)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("RecordTransaction() error = %v, se esperaba %v", err, tt.wantErr)
				}
				if len(transactions.stored) != 0 {
					t.Error("no debe persistirse una transacción inválida")
				}
				return
			}
			if err != nil {
				t.Fatalf("RecordTransaction() error inesperado: %v", err)
			}
			if tx.ID.IsZero() || tx.UserID != tt.userID || len(transactions.stored) != 1 {
				t.Errorf("la transacción no quedó persistida para el usuario: %+v", tx)
			}
			tt.check(t, tx)
		})
	}
}

func TestFinanceServiceListTransactions(t *testing.T) {
	errDB := errors.New("base caída")
	tests := []struct {
		name      string
		filter    finance.TransactionFilter
		listErr   error
		wantErr   error
		wantLimit int
	}{
		{name: "límite por defecto", wantLimit: 50},
		{name: "respeta un límite válido", filter: finance.TransactionFilter{Limit: 10}, wantLimit: 10},
		{name: "recorta al máximo", filter: finance.TransactionFilter{Limit: 500}, wantLimit: 200},
		{name: "rango invertido", filter: finance.TransactionFilter{From: fixedNow, To: fixedNow.Add(-time.Hour)}, wantErr: domain.ErrInvalidInput},
		{name: "rango vacío", filter: finance.TransactionFilter{From: fixedNow, To: fixedNow}, wantErr: domain.ErrInvalidInput},
		{name: "offset negativo", filter: finance.TransactionFilter{Offset: -1}, wantErr: domain.ErrInvalidInput},
		{name: "tipo inválido", filter: finance.TransactionFilter{Type: "transfer"}, wantErr: domain.ErrInvalidInput},
		{name: "error del repositorio", listErr: errDB, wantErr: errDB},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transactions := &fakeTransactionRepo{listErr: tt.listErr}
			svc := NewFinanceService(newFakeCategoryRepo(), transactions, newFakeUserRepo(), fixedClock)

			_, err := svc.ListTransactions(context.Background(), domain.NewID(), tt.filter)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ListTransactions() error = %v, se esperaba %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ListTransactions() error inesperado: %v", err)
			}
			if transactions.lastFilter.Limit != tt.wantLimit {
				t.Errorf("Limit enviado = %d, se esperaba %d", transactions.lastFilter.Limit, tt.wantLimit)
			}
		})
	}
}

func TestFinanceServiceBalance(t *testing.T) {
	owner, other := domain.NewID(), domain.NewID()
	errDB := errors.New("timeout")
	october := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	november := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	tx := func(userID domain.ID, typ finance.EntryType, cents int64, currency domain.Currency, at time.Time) finance.Transaction {
		return finance.Transaction{UserID: userID, Type: typ, Amount: finance.Money{Cents: cents, Currency: currency}, OccurredAt: at}
	}
	seed := []finance.Transaction{
		tx(owner, finance.Income, 100000, "USD", october.AddDate(0, 0, 1)),
		tx(owner, finance.Expense, 2599, "USD", october.AddDate(0, 0, 5)),
		tx(owner, finance.Expense, 500000, "CRC", october.AddDate(0, 0, 9)),
		tx(owner, finance.Expense, 999, "USD", november),                      // fuera del rango [oct, nov)
		tx(other, finance.Income, 777, "USD", october.AddDate(0, 0, 2)),       // de otro usuario
		tx(owner, finance.Income, 5000, "USD", october.Add(-time.Nanosecond)), // fuera del rango
	}

	tests := []struct {
		name     string
		from, to time.Time
		listErr  error
		wantErr  error
		want     []finance.Balance
	}{
		{
			name: "octubre por moneda",
			from: october, to: november,
			want: []finance.Balance{
				{Currency: "CRC", ExpenseCents: 500000},
				{Currency: "USD", IncomeCents: 100000, ExpenseCents: 2599},
			},
		},
		{
			name: "sin límites incluye todo lo del usuario",
			want: []finance.Balance{
				{Currency: "CRC", ExpenseCents: 500000},
				{Currency: "USD", IncomeCents: 105000, ExpenseCents: 3598},
			},
		},
		{name: "periodo sin movimientos", from: november.AddDate(0, 1, 0), to: november.AddDate(0, 2, 0), want: []finance.Balance{}},
		{name: "rango invertido", from: november, to: october, wantErr: domain.ErrInvalidInput},
		{name: "error del repositorio", listErr: errDB, wantErr: errDB},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transactions := &fakeTransactionRepo{stored: seed, listErr: tt.listErr}
			svc := NewFinanceService(newFakeCategoryRepo(), transactions, newFakeUserRepo(), fixedClock)

			got, err := svc.Balance(context.Background(), owner, tt.from, tt.to)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Balance() error = %v, se esperaba %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Balance() error inesperado: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("Balance() = %+v, se esperaba %+v", got, tt.want)
			}
			if transactions.lastFilter.Limit != 0 {
				t.Errorf("Balance no debe paginar: Limit = %d", transactions.lastFilter.Limit)
			}
		})
	}
}

func TestFinanceServiceUpdateTransaction(t *testing.T) {
	owner := existingUser(t, "ana@example.com", "UTC")
	stranger := existingUser(t, "otro@example.com", "UTC")
	food := finance.Category{ID: domain.NewID(), UserID: owner.ID, Name: "Comida", Type: finance.Expense}
	salary := finance.Category{ID: domain.NewID(), UserID: owner.ID, Name: "Salario", Type: finance.Income}
	original := finance.Transaction{
		ID: domain.NewID(), UserID: owner.ID, CategoryID: &food.ID, Type: finance.Expense,
		Amount: finance.Money{Cents: 2500, Currency: "CRC"}, Description: "Almuerzo", OccurredAt: fixedNow,
	}
	cents := func(n int64) *int64 { return &n }
	str := func(s string) *string { return &s }
	entry := func(e finance.EntryType) *finance.EntryType { return &e }
	yesterday := fixedNow.Add(-24 * time.Hour)

	tests := []struct {
		name    string
		userID  domain.ID
		in      UpdateTransactionInput
		wantErr error
		check   func(t *testing.T, tx finance.Transaction)
	}{
		{
			name:   "corrige monto, descripción y fecha",
			userID: owner.ID,
			in:     UpdateTransactionInput{AmountCents: cents(3000), Description: str(" Almuerzo y café "), OccurredAt: &yesterday},
			check: func(t *testing.T, tx finance.Transaction) {
				if tx.Amount.Cents != 3000 || tx.Description != "Almuerzo y café" || !tx.OccurredAt.Equal(yesterday) {
					t.Errorf("transacción = %+v", tx)
				}
				if tx.CategoryID == nil || *tx.CategoryID != food.ID || tx.Amount.Currency != "CRC" {
					t.Errorf("los campos no enviados no debían cambiar: %+v", tx)
				}
			},
		},
		{
			name:   "cambia a ingreso con una categoría de ingresos",
			userID: owner.ID,
			in:     UpdateTransactionInput{Type: entry(finance.Income), SetCategory: true, CategoryID: &salary.ID},
			check: func(t *testing.T, tx finance.Transaction) {
				if tx.Type != finance.Income || tx.CategoryID == nil || *tx.CategoryID != salary.ID {
					t.Errorf("transacción = %+v", tx)
				}
			},
		},
		{
			name:   "quita la categoría",
			userID: owner.ID,
			in:     UpdateTransactionInput{SetCategory: true},
			check: func(t *testing.T, tx finance.Transaction) {
				if tx.CategoryID != nil {
					t.Errorf("CategoryID = %v, se esperaba nil", *tx.CategoryID)
				}
			},
		},
		{
			name:   "cambia la moneda",
			userID: owner.ID,
			in:     UpdateTransactionInput{Currency: str("usd")},
			check: func(t *testing.T, tx finance.Transaction) {
				if tx.Amount.Currency != "USD" {
					t.Errorf("Currency = %s", tx.Amount.Currency)
				}
			},
		},
		{name: "cambiar el tipo sin cambiar una categoría incompatible", userID: owner.ID, in: UpdateTransactionInput{Type: entry(finance.Income)}, wantErr: domain.ErrInvalidInput},
		{name: "monto cero", userID: owner.ID, in: UpdateTransactionInput{AmountCents: cents(0)}, wantErr: domain.ErrInvalidInput},
		{name: "moneda inválida", userID: owner.ID, in: UpdateTransactionInput{Currency: str("XX")}, wantErr: domain.ErrInvalidInput},
		{name: "categoría inexistente", userID: owner.ID, in: UpdateTransactionInput{SetCategory: true, CategoryID: new(domain.NewID())}, wantErr: domain.ErrNotFound},
		{name: "transacción de otro usuario", userID: stranger.ID, in: UpdateTransactionInput{AmountCents: cents(1)}, wantErr: domain.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transactions := &fakeTransactionRepo{stored: []finance.Transaction{original}}
			svc := NewFinanceService(newFakeCategoryRepo(food, salary), transactions, newFakeUserRepo(owner, stranger), fixedClock)

			_, err := svc.UpdateTransaction(context.Background(), tt.userID, original.ID, tt.in)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("UpdateTransaction() error = %v, se esperaba %v", err, tt.wantErr)
				}
				if stored := transactions.stored[0]; stored.Amount != original.Amount || stored.Type != original.Type {
					t.Errorf("un cambio inválido no debe persistirse: %+v", stored)
				}
				return
			}
			if err != nil {
				t.Fatalf("UpdateTransaction() error inesperado: %v", err)
			}
			tt.check(t, transactions.stored[0])
		})
	}
}

func TestFinanceServiceUpdateTransactionKeepsDeletedCategory(t *testing.T) {
	// Si la categoría se borró después, corregir el monto no debe fallar por ella.
	owner := existingUser(t, "ana@example.com", "UTC")
	deletedCategory := domain.NewID()
	tx := finance.Transaction{
		ID: domain.NewID(), UserID: owner.ID, CategoryID: &deletedCategory, Type: finance.Expense,
		Amount: finance.Money{Cents: 100, Currency: "CRC"}, OccurredAt: fixedNow,
	}
	transactions := &fakeTransactionRepo{stored: []finance.Transaction{tx}}
	svc := NewFinanceService(newFakeCategoryRepo(), transactions, newFakeUserRepo(owner), fixedClock)

	amount := int64(250)
	if _, err := svc.UpdateTransaction(context.Background(), owner.ID, tx.ID, UpdateTransactionInput{AmountCents: &amount}); err != nil {
		t.Fatalf("UpdateTransaction() error inesperado: %v", err)
	}
	if got := transactions.stored[0]; got.Amount.Cents != 250 || got.CategoryID == nil {
		t.Errorf("transacción = %+v", got)
	}
}

func TestFinanceServiceDeleteTransaction(t *testing.T) {
	owner := existingUser(t, "ana@example.com", "UTC")
	stranger := existingUser(t, "otro@example.com", "UTC")
	tx := finance.Transaction{ID: domain.NewID(), UserID: owner.ID, Type: finance.Expense, Amount: finance.Money{Cents: 100, Currency: "CRC"}, OccurredAt: fixedNow}

	tests := []struct {
		name    string
		userID  domain.ID
		wantErr error
	}{
		{name: "elimina la propia", userID: owner.ID},
		{name: "transacción de otro usuario", userID: stranger.ID, wantErr: domain.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transactions := &fakeTransactionRepo{stored: []finance.Transaction{tx}}
			svc := NewFinanceService(newFakeCategoryRepo(), transactions, newFakeUserRepo(owner, stranger), fixedClock)

			err := svc.DeleteTransaction(context.Background(), tt.userID, tx.ID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("DeleteTransaction() error = %v, se esperaba %v", err, tt.wantErr)
			}
			wantRemaining := 1
			if tt.wantErr == nil {
				wantRemaining = 0
			}
			if remaining := len(transactions.stored); remaining != wantRemaining {
				t.Errorf("quedan %d transacciones, se esperaban %d", remaining, wantRemaining)
			}
		})
	}
}
