// Package finance modela las finanzas personales: categorías, transacciones y montos.
package finance

import (
	"fmt"

	"bytebalance/internal/domain"
)

// EntryType distingue ingresos de gastos; aplica a categorías y transacciones.
type EntryType string

// Tipos de movimiento admitidos.
const (
	Income  EntryType = "income"
	Expense EntryType = "expense"
)

// Valid indica si t es un tipo de movimiento conocido.
func (t EntryType) Valid() bool {
	return t == Income || t == Expense
}

// MaxCents es el mayor monto representable por NUMERIC(14,2): 999.999.999.999,99.
const MaxCents int64 = 99_999_999_999_999

// Money es un monto en unidades mínimas (centavos) de una moneda concreta.
// Se usan enteros para evitar errores de redondeo de punto flotante.
type Money struct {
	Cents    int64
	Currency domain.Currency
}

// Validate comprueba que el monto sea positivo, quepa en la base de datos y tenga moneda válida.
func (m Money) Validate() error {
	if m.Cents <= 0 || m.Cents > MaxCents {
		return fmt.Errorf("%w: el monto debe ser mayor que cero y no superar %d centavos", domain.ErrInvalidInput, MaxCents)
	}
	if !m.Currency.Valid() {
		return fmt.Errorf("%w: moneda %q inválida", domain.ErrInvalidInput, m.Currency)
	}
	return nil
}
