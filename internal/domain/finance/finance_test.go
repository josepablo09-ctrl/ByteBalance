package finance

import (
	"errors"
	"strings"
	"testing"
	"time"

	"bytebalance/internal/domain"
)

const (
	userA domain.ID = "11111111-1111-1111-1111-111111111111"
	userB domain.ID = "22222222-2222-2222-2222-222222222222"
)

func TestMoneyValidate(t *testing.T) {
	tests := []struct {
		name    string
		money   Money
		wantErr bool
	}{
		{name: "válido", money: Money{Cents: 1050, Currency: "USD"}},
		{name: "máximo", money: Money{Cents: MaxCents, Currency: "USD"}},
		{name: "cero", money: Money{Cents: 0, Currency: "USD"}, wantErr: true},
		{name: "negativo", money: Money{Cents: -1, Currency: "USD"}, wantErr: true},
		{name: "excede NUMERIC(14,2)", money: Money{Cents: MaxCents + 1, Currency: "USD"}, wantErr: true},
		{name: "sin moneda", money: Money{Cents: 100}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.money.Validate()
			if tt.wantErr != (err != nil) {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("Validate() error = %v, se esperaba ErrInvalidInput", err)
			}
		})
	}
}

func TestNewCategory(t *testing.T) {
	tests := []struct {
		name     string
		userID   domain.ID
		catName  string
		catType  EntryType
		wantErr  bool
		wantName string
	}{
		{name: "válida", userID: userA, catName: " Comida ", catType: Expense, wantName: "Comida"},
		{name: "sin usuario", catName: "Comida", catType: Expense, wantErr: true},
		{name: "nombre vacío", userID: userA, catName: "  ", catType: Expense, wantErr: true},
		{name: "nombre demasiado largo", userID: userA, catName: strings.Repeat("x", 81), catType: Expense, wantErr: true},
		{name: "tipo inválido", userID: userA, catName: "Comida", catType: "transfer", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewCategory(tt.userID, tt.catName, tt.catType)
			if tt.wantErr {
				if !errors.Is(err, domain.ErrInvalidInput) {
					t.Fatalf("NewCategory() error = %v, se esperaba ErrInvalidInput", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewCategory() error inesperado: %v", err)
			}
			if c.Name != tt.wantName {
				t.Errorf("Name = %q, se esperaba %q", c.Name, tt.wantName)
			}
		})
	}
}

func TestNewTransaction(t *testing.T) {
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	deleted := now
	food := &Category{ID: "c1", UserID: userA, Name: "Comida", Type: Expense}

	tests := []struct {
		name    string
		modify  func(p *TransactionParams)
		wantErr error
	}{
		{name: "válida sin categoría", modify: func(*TransactionParams) {}},
		{name: "válida con categoría", modify: func(p *TransactionParams) { p.Category = food }},
		{name: "sin usuario", modify: func(p *TransactionParams) { p.UserID = "" }, wantErr: domain.ErrInvalidInput},
		{name: "tipo inválido", modify: func(p *TransactionParams) { p.Type = "x" }, wantErr: domain.ErrInvalidInput},
		{name: "monto cero", modify: func(p *TransactionParams) { p.Amount.Cents = 0 }, wantErr: domain.ErrInvalidInput},
		{name: "sin fecha", modify: func(p *TransactionParams) { p.OccurredAt = time.Time{} }, wantErr: domain.ErrInvalidInput},
		{name: "descripción larga", modify: func(p *TransactionParams) { p.Description = strings.Repeat("a", 256) }, wantErr: domain.ErrInvalidInput},
		{
			name:    "categoría de otro tipo",
			modify:  func(p *TransactionParams) { p.Category = &Category{ID: "c2", UserID: userA, Type: Income} },
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:    "categoría de otro usuario",
			modify:  func(p *TransactionParams) { p.Category = &Category{ID: "c3", UserID: userB, Type: Expense} },
			wantErr: domain.ErrNotFound,
		},
		{
			name: "categoría eliminada",
			modify: func(p *TransactionParams) {
				p.Category = &Category{ID: "c4", UserID: userA, Type: Expense, DeletedAt: &deleted}
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:    "categoría sin persistir",
			modify:  func(p *TransactionParams) { p.Category = &Category{UserID: userA, Type: Expense} },
			wantErr: domain.ErrInvalidInput,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := TransactionParams{
				UserID:      userA,
				Type:        Expense,
				Amount:      Money{Cents: 2599, Currency: "USD"},
				Description: " Almuerzo ",
				OccurredAt:  now,
			}
			tt.modify(&p)
			tx, err := NewTransaction(p)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("NewTransaction() error = %v, se esperaba %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewTransaction() error inesperado: %v", err)
			}
			if tx.Description != "Almuerzo" {
				t.Errorf("Description = %q, se esperaba sin espacios", tx.Description)
			}
			if p.Category != nil && (tx.CategoryID == nil || *tx.CategoryID != p.Category.ID) {
				t.Errorf("CategoryID = %v, se esperaba %q", tx.CategoryID, p.Category.ID)
			}
		})
	}
}

func TestSignedCents(t *testing.T) {
	tests := []struct {
		name  string
		typ   EntryType
		cents int64
		want  int64
	}{
		{name: "ingreso positivo", typ: Income, cents: 500, want: 500},
		{name: "gasto negativo", typ: Expense, cents: 500, want: -500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := Transaction{Type: tt.typ, Amount: Money{Cents: tt.cents, Currency: "USD"}}
			if got := tx.SignedCents(); got != tt.want {
				t.Errorf("SignedCents() = %d, se esperaba %d", got, tt.want)
			}
		})
	}
}
