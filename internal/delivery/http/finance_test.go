package http

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/finance"
	"bytebalance/internal/usecase"
)

func TestCreateCategory(t *testing.T) {
	created := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		body       string
		createFn   func(domain.ID, string, finance.EntryType) (*finance.Category, error)
		wantStatus int
	}{
		{
			name: "categoría válida",
			body: `{"name":"Comida","type":"expense"}`,
			createFn: func(id domain.ID, name string, typ finance.EntryType) (*finance.Category, error) {
				if id != testUserID || name != "Comida" || typ != finance.Expense {
					t.Errorf("CreateCategory(%v, %q, %q)", id, name, typ)
				}
				return &finance.Category{ID: domain.NewID(), UserID: id, Name: name, Type: typ, CreatedAt: created}, nil
			},
			wantStatus: http.StatusCreated,
		},
		{name: "falta type", body: `{"name":"Comida"}`, wantStatus: http.StatusBadRequest},
		{
			name: "tipo inválido según el dominio",
			body: `{"name":"Comida","type":"transfer"}`,
			createFn: func(domain.ID, string, finance.EntryType) (*finance.Category, error) {
				return nil, fmt.Errorf("crear categoría: %w: tipo inválido", domain.ErrInvalidInput)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestRouter(Services{Finance: stubFinance{createCategory: tt.createFn}})
			rec := do(h, http.MethodPost, "/api/v1/categories", tt.body, true)
			expectStatus(t, rec, tt.wantStatus)
		})
	}
}

func TestListCategoriesEmptyIsArray(t *testing.T) {
	h := newTestRouter(Services{Finance: stubFinance{
		listCategories: func(domain.ID) ([]finance.Category, error) { return nil, nil },
	}})
	rec := do(h, http.MethodGet, "/api/v1/categories", "", true)
	expectStatus(t, rec, http.StatusOK)
	if got := rec.Body.String(); got != "{\"categories\":[]}\n" {
		t.Errorf("body = %q, se esperaba una lista vacía y no null", got)
	}
}

func TestRecordTransaction(t *testing.T) {
	categoryID := domain.NewID()
	occurredAt := time.Date(2026, 10, 10, 13, 30, 0, 0, time.UTC)
	tests := []struct {
		name       string
		body       string
		recordFn   func(domain.ID, usecase.RecordTransactionInput) (*finance.Transaction, error)
		wantStatus int
	}{
		{
			name: "gasto completo",
			body: fmt.Sprintf(`{"type":"expense","amount_cents":2599,"currency":"USD","category_id":%q,"description":"Almuerzo","occurred_at":"2026-10-10T13:30:00Z"}`, categoryID),
			recordFn: func(id domain.ID, in usecase.RecordTransactionInput) (*finance.Transaction, error) {
				if id != testUserID || in.AmountCents != 2599 || in.CategoryID == nil || *in.CategoryID != categoryID || !in.OccurredAt.Equal(occurredAt) {
					t.Errorf("RecordTransaction(%v, %+v)", id, in)
				}
				return &finance.Transaction{
					ID: domain.NewID(), UserID: id, CategoryID: in.CategoryID, Type: in.Type,
					Amount: finance.Money{Cents: in.AmountCents, Currency: "USD"}, OccurredAt: in.OccurredAt,
				}, nil
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "campos opcionales ausentes",
			body: `{"type":"income","amount_cents":100}`,
			recordFn: func(_ domain.ID, in usecase.RecordTransactionInput) (*finance.Transaction, error) {
				if in.CategoryID != nil || !in.OccurredAt.IsZero() || in.Currency != "" {
					t.Errorf("los opcionales ausentes deben llegar vacíos: %+v", in)
				}
				return &finance.Transaction{ID: domain.NewID(), Type: in.Type, Amount: finance.Money{Cents: 100, Currency: "CRC"}}, nil
			},
			wantStatus: http.StatusCreated,
		},
		{name: "falta amount_cents", body: `{"type":"expense"}`, wantStatus: http.StatusBadRequest},
		{name: "monto decimal", body: `{"type":"expense","amount_cents":25.99}`, wantStatus: http.StatusBadRequest},
		{name: "category_id inválido", body: `{"type":"expense","amount_cents":100,"category_id":"no-uuid"}`, wantStatus: http.StatusBadRequest},
		{name: "fecha inválida", body: `{"type":"expense","amount_cents":100,"occurred_at":"10/10/2026"}`, wantStatus: http.StatusBadRequest},
		{
			name: "monto cero llega al dominio",
			body: `{"type":"expense","amount_cents":0}`,
			recordFn: func(domain.ID, usecase.RecordTransactionInput) (*finance.Transaction, error) {
				return nil, fmt.Errorf("registrar transacción: %w: el monto debe ser mayor que cero", domain.ErrInvalidInput)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "categoría ajena",
			body: fmt.Sprintf(`{"type":"expense","amount_cents":100,"category_id":%q}`, categoryID),
			recordFn: func(domain.ID, usecase.RecordTransactionInput) (*finance.Transaction, error) {
				return nil, fmt.Errorf("registrar transacción: categoría: %w", domain.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestRouter(Services{Finance: stubFinance{recordTransaction: tt.recordFn}})
			rec := do(h, http.MethodPost, "/api/v1/transactions", tt.body, true)
			expectStatus(t, rec, tt.wantStatus)
			if tt.wantStatus == http.StatusCreated {
				if body := decodeJSONBody(t, rec); body["amount_cents"] == nil || body["currency"] == nil {
					t.Errorf("respuesta = %v", body)
				}
			}
		})
	}
}

func TestListTransactions(t *testing.T) {
	categoryID := domain.NewID()
	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantFilter finance.TransactionFilter
	}{
		{name: "sin filtros", query: "", wantStatus: http.StatusOK},
		{
			name:       "todos los filtros",
			query:      "?from=2026-10-01T00:00:00Z&to=2026-11-01T00:00:00-06:00&type=expense&category_id=" + categoryID.String() + "&limit=10&offset=20",
			wantStatus: http.StatusOK,
			wantFilter: finance.TransactionFilter{
				From:       time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
				To:         time.Date(2026, 11, 1, 6, 0, 0, 0, time.UTC),
				Type:       finance.Expense,
				CategoryID: &categoryID,
				Limit:      10,
				Offset:     20,
			},
		},
		{name: "from inválido", query: "?from=2026-10-01", wantStatus: http.StatusBadRequest},
		{name: "limit no numérico", query: "?limit=diez", wantStatus: http.StatusBadRequest},
		{name: "category_id inválido", query: "?category_id=123", wantStatus: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got finance.TransactionFilter
			h := newTestRouter(Services{Finance: stubFinance{
				listTransactions: func(_ domain.ID, f finance.TransactionFilter) ([]finance.Transaction, error) {
					got = f
					return nil, nil
				},
			}})
			rec := do(h, http.MethodGet, "/api/v1/transactions"+tt.query, "", true)
			expectStatus(t, rec, tt.wantStatus)
			if tt.wantStatus != http.StatusOK {
				return
			}
			sameCategory := (got.CategoryID == nil) == (tt.wantFilter.CategoryID == nil) &&
				(got.CategoryID == nil || *got.CategoryID == *tt.wantFilter.CategoryID)
			if !got.From.Equal(tt.wantFilter.From) || !got.To.Equal(tt.wantFilter.To) || got.Type != tt.wantFilter.Type ||
				!sameCategory || got.Limit != tt.wantFilter.Limit || got.Offset != tt.wantFilter.Offset {
				t.Errorf("filtro = %+v, se esperaba %+v", got, tt.wantFilter)
			}
		})
	}
}

func TestBalance(t *testing.T) {
	h := newTestRouter(Services{Finance: stubFinance{
		balance: func(id domain.ID, from, to time.Time) ([]finance.Balance, error) {
			if id != testUserID || from.IsZero() || !to.IsZero() {
				t.Errorf("Balance(%v, %v, %v)", id, from, to)
			}
			return []finance.Balance{{Currency: "USD", IncomeCents: 100000, ExpenseCents: 2599}}, nil
		},
	}})
	rec := do(h, http.MethodGet, "/api/v1/balance?from=2026-10-01T00:00:00Z", "", true)
	expectStatus(t, rec, http.StatusOK)
	want := "{\"balances\":[{\"currency\":\"USD\",\"income_cents\":100000,\"expense_cents\":2599,\"net_cents\":97401}]}\n"
	if got := rec.Body.String(); got != want {
		t.Errorf("body = %s, se esperaba %s", got, want)
	}

	if rec := do(h, http.MethodGet, "/api/v1/balance?to=ayer", "", true); rec.Code != http.StatusBadRequest {
		t.Errorf("to inválido: status = %d, se esperaba 400", rec.Code)
	}
}

func TestUpdateTransaction(t *testing.T) {
	txID := domain.NewID()
	categoryID := domain.NewID()
	path := "/api/v1/transactions/" + txID.String()
	echo := func(check func(usecase.UpdateTransactionInput)) func(domain.ID, domain.ID, usecase.UpdateTransactionInput) (*finance.Transaction, error) {
		return func(userID, gotID domain.ID, in usecase.UpdateTransactionInput) (*finance.Transaction, error) {
			if userID != testUserID || gotID != txID {
				t.Errorf("UpdateTransaction(%v, %v)", userID, gotID)
			}
			check(in)
			tx := &finance.Transaction{ID: txID, Type: finance.Expense, Amount: finance.Money{Cents: 100, Currency: "CRC"}, CategoryID: in.CategoryID}
			if in.AmountCents != nil {
				tx.Amount.Cents = *in.AmountCents
			}
			return tx, nil
		}
	}

	tests := []struct {
		name       string
		target     string
		body       string
		updateFn   func(domain.ID, domain.ID, usecase.UpdateTransactionInput) (*finance.Transaction, error)
		wantStatus int
		wantBody   map[string]any
	}{
		{
			name:   "corrige el monto",
			target: path,
			body:   `{"amount_cents":4550}`,
			updateFn: echo(func(in usecase.UpdateTransactionInput) {
				if in.AmountCents == nil || *in.AmountCents != 4550 || in.SetCategory || in.Type != nil {
					t.Errorf("UpdateTransactionInput = %+v", in)
				}
			}),
			wantStatus: http.StatusOK,
			wantBody:   map[string]any{"amount_cents": float64(4550)},
		},
		{
			name:   "cambia tipo, categoría y fecha",
			target: path,
			body:   `{"type":"income","category_id":"` + categoryID.String() + `","occurred_at":"2026-10-01T18:00:00Z"}`,
			updateFn: echo(func(in usecase.UpdateTransactionInput) {
				if in.Type == nil || *in.Type != finance.Income || !in.SetCategory || in.CategoryID == nil || *in.CategoryID != categoryID {
					t.Errorf("UpdateTransactionInput = %+v", in)
				}
				if in.OccurredAt == nil || !in.OccurredAt.Equal(time.Date(2026, 10, 1, 18, 0, 0, 0, time.UTC)) {
					t.Errorf("OccurredAt = %v", in.OccurredAt)
				}
			}),
			wantStatus: http.StatusOK,
			wantBody:   map[string]any{"category_id": categoryID.String()},
		},
		{
			name:   "null quita la categoría",
			target: path,
			body:   `{"category_id":null}`,
			updateFn: echo(func(in usecase.UpdateTransactionInput) {
				if !in.SetCategory || in.CategoryID != nil {
					t.Errorf("UpdateTransactionInput = %+v", in)
				}
			}),
			wantStatus: http.StatusOK,
			wantBody:   map[string]any{"category_id": nil},
		},
		{name: "sin campos", target: path, body: `{}`, wantStatus: http.StatusBadRequest},
		{name: "categoría no UUID", target: path, body: `{"category_id":"comida"}`, wantStatus: http.StatusBadRequest},
		{name: "fecha inválida", target: path, body: `{"occurred_at":"ayer"}`, wantStatus: http.StatusBadRequest},
		{name: "id inválido", target: "/api/v1/transactions/no-uuid", body: `{"amount_cents":1}`, wantStatus: http.StatusBadRequest},
		{
			name:   "valor inválido para el dominio",
			target: path,
			body:   `{"amount_cents":0}`,
			updateFn: func(domain.ID, domain.ID, usecase.UpdateTransactionInput) (*finance.Transaction, error) {
				return nil, fmt.Errorf("actualizar transacción: %w: el monto debe ser mayor que cero", domain.ErrInvalidInput)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:   "transacción de otro usuario",
			target: path,
			body:   `{"amount_cents":1}`,
			updateFn: func(domain.ID, domain.ID, usecase.UpdateTransactionInput) (*finance.Transaction, error) {
				return nil, fmt.Errorf("actualizar transacción: %w", domain.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestRouter(Services{Finance: stubFinance{updateTransaction: tt.updateFn}})
			rec := do(h, http.MethodPatch, tt.target, tt.body, true)
			expectStatus(t, rec, tt.wantStatus)
			if tt.wantBody == nil {
				return
			}
			body := decodeJSONBody(t, rec)
			for k, v := range tt.wantBody {
				if body[k] != v {
					t.Errorf("%s = %v, se esperaba %v", k, body[k], v)
				}
			}
		})
	}
}

func TestDeleteTransaction(t *testing.T) {
	txID := domain.NewID()
	tests := []struct {
		name       string
		target     string
		deleteErr  error
		wantStatus int
	}{
		{name: "elimina", target: "/api/v1/transactions/" + txID.String(), wantStatus: http.StatusNoContent},
		{name: "inexistente", target: "/api/v1/transactions/" + txID.String(), deleteErr: fmt.Errorf("eliminar transacción: %w", domain.ErrNotFound), wantStatus: http.StatusNotFound},
		{name: "id inválido", target: "/api/v1/transactions/no-uuid", wantStatus: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestRouter(Services{Finance: stubFinance{deleteTransaction: func(userID, gotID domain.ID) error {
				if userID != testUserID || gotID != txID {
					t.Errorf("DeleteTransaction(%v, %v)", userID, gotID)
				}
				return tt.deleteErr
			}}})
			rec := do(h, http.MethodDelete, tt.target, "", true)
			expectStatus(t, rec, tt.wantStatus)
		})
	}
}
