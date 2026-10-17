package finance

import (
	"slices"
	"testing"

	"bytebalance/internal/domain"
)

func TestSummarize(t *testing.T) {
	tx := func(typ EntryType, cents int64, currency string) Transaction {
		return Transaction{Type: typ, Amount: Money{Cents: cents, Currency: domain.Currency(currency)}}
	}
	tests := []struct {
		name string
		txs  []Transaction
		want []Balance
	}{
		{name: "sin transacciones", want: []Balance{}},
		{
			name: "una moneda",
			txs:  []Transaction{tx(Income, 100000, "USD"), tx(Expense, 2599, "USD"), tx(Expense, 1, "USD")},
			want: []Balance{{Currency: "USD", IncomeCents: 100000, ExpenseCents: 2600}},
		},
		{
			name: "aportes y retiros de ahorro",
			txs: []Transaction{
				tx(Income, 750_000_00, "CRC"), tx(Expense, 275_000_00, "CRC"),
				tx(Saving, 100_000_00, "CRC"), tx(Saving, 50_000_00, "CRC"), tx(Withdrawal, 30_000_00, "CRC"),
			},
			want: []Balance{{Currency: "CRC", IncomeCents: 750_000_00, ExpenseCents: 275_000_00, SavedCents: 150_000_00, WithdrawnCents: 30_000_00}},
		},
		{
			name: "varias monedas ordenadas y sin mezclar",
			txs:  []Transaction{tx(Expense, 500, "USD"), tx(Income, 900000, "CRC"), tx(Income, 700, "USD")},
			want: []Balance{
				{Currency: "CRC", IncomeCents: 900000},
				{Currency: "USD", IncomeCents: 700, ExpenseCents: 500},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Summarize(tt.txs)
			if !slices.Equal(got, tt.want) {
				t.Errorf("Summarize() = %+v, se esperaba %+v", got, tt.want)
			}
		})
	}
}

func TestBalanceFormulas(t *testing.T) {
	tests := []struct {
		name                         string
		balance                      Balance
		wantNet, wantSaved, wantFree int64
	}{
		{name: "positivo sin ahorro", balance: Balance{IncomeCents: 1000, ExpenseCents: 250}, wantNet: 750, wantFree: 750},
		{name: "negativo sin ahorro", balance: Balance{IncomeCents: 100, ExpenseCents: 250}, wantNet: -150, wantFree: -150},
		{
			// 750 000 - 275 000 = 475 000 de balance; 120 000 apartados → 355 000 disponibles.
			name:    "el ahorro no cambia el balance pero sí el disponible",
			balance: Balance{IncomeCents: 750_000_00, ExpenseCents: 275_000_00, SavedCents: 150_000_00, WithdrawnCents: 30_000_00},
			wantNet: 475_000_00, wantSaved: 120_000_00, wantFree: 355_000_00,
		},
		{
			name:    "apartar más de lo disponible deja el disponible en negativo",
			balance: Balance{IncomeCents: 1000, ExpenseCents: 800, SavedCents: 500},
			wantNet: 200, wantSaved: 500, wantFree: -300,
		},
		{
			name:    "solo retiros en el periodo: ahorro del periodo negativo",
			balance: Balance{WithdrawnCents: 400},
			wantNet: 0, wantSaved: -400, wantFree: 400,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := tt.balance
			if got := b.NetCents(); got != tt.wantNet {
				t.Errorf("NetCents() = %d, se esperaba %d", got, tt.wantNet)
			}
			if got := b.SavingsCents(); got != tt.wantSaved {
				t.Errorf("SavingsCents() = %d, se esperaba %d", got, tt.wantSaved)
			}
			if got := b.AvailableCents(); got != tt.wantFree {
				t.Errorf("AvailableCents() = %d, se esperaba %d", got, tt.wantFree)
			}
			// Invariante: balance total = disponible + ahorro.
			if b.NetCents() != b.AvailableCents()+b.SavingsCents() {
				t.Error("el balance total debe ser la suma del disponible y el ahorro")
			}
		})
	}
}
