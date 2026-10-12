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

func TestBalanceNetCents(t *testing.T) {
	tests := []struct {
		name    string
		balance Balance
		want    int64
	}{
		{name: "positivo", balance: Balance{IncomeCents: 1000, ExpenseCents: 250}, want: 750},
		{name: "negativo", balance: Balance{IncomeCents: 100, ExpenseCents: 250}, want: -150},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.balance.NetCents(); got != tt.want {
				t.Errorf("NetCents() = %d, se esperaba %d", got, tt.want)
			}
		})
	}
}
