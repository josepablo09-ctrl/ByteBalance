// Package finance modela las finanzas personales: categorías, transacciones y montos.
package finance

import (
	"fmt"

	"bytebalance/internal/domain"
)

// EntryType es el tipo de una transacción. Las categorías solo pueden ser de
// ingresos o gastos (ver Categorizable).
type EntryType string

// Tipos de movimiento admitidos.
const (
	Income  EntryType = "income"
	Expense EntryType = "expense"
	// Saving aparta dinero: deja de estar disponible para gastar y pasa al ahorro.
	Saving EntryType = "saving"
	// Withdrawal retira dinero del ahorro y lo devuelve al disponible.
	Withdrawal EntryType = "withdrawal"
)

// Valid indica si t es un tipo de movimiento conocido.
func (t EntryType) Valid() bool {
	return t.Categorizable() || t.IsSavings()
}

// Categorizable indica si t admite categorías: solo ingresos y gastos.
func (t EntryType) Categorizable() bool {
	return t == Income || t == Expense
}

// IsSavings indica si t mueve dinero entre el disponible y el ahorro. Estos
// movimientos no cambian el balance total.
func (t EntryType) IsSavings() bool {
	return t == Saving || t == Withdrawal
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
