package finance

import (
	"slices"
	"strings"

	"bytebalance/internal/domain"
)

// Balance resume los ingresos y gastos de una moneda.
type Balance struct {
	Currency     domain.Currency
	IncomeCents  int64
	ExpenseCents int64
}

// NetCents devuelve ingresos menos gastos.
func (b Balance) NetCents() int64 {
	return b.IncomeCents - b.ExpenseCents
}

// Summarize agrupa las transacciones por moneda y devuelve un Balance por cada
// una, ordenado por código de moneda. Las monedas nunca se mezclan, porque no
// hay conversión entre ellas.
func Summarize(transactions []Transaction) []Balance {
	byCurrency := map[domain.Currency]*Balance{}
	for _, t := range transactions {
		b, ok := byCurrency[t.Amount.Currency]
		if !ok {
			b = &Balance{Currency: t.Amount.Currency}
			byCurrency[t.Amount.Currency] = b
		}
		switch t.Type {
		case Income:
			b.IncomeCents += t.Amount.Cents
		case Expense:
			b.ExpenseCents += t.Amount.Cents
		}
	}

	balances := make([]Balance, 0, len(byCurrency))
	for _, b := range byCurrency {
		balances = append(balances, *b)
	}
	slices.SortFunc(balances, func(a, b Balance) int {
		return strings.Compare(string(a.Currency), string(b.Currency))
	})
	return balances
}
