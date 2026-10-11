// Package domain contiene los tipos y errores compartidos por todos los módulos
// de dominio de ByteBalance (user, finance, habit). Solo depende de la stdlib.
package domain

import (
	"errors"
	"fmt"
	"strings"
)

// Errores de dominio. La capa delivery los traduce a códigos HTTP con errors.Is.
var (
	// ErrNotFound indica que la entidad solicitada no existe (o no pertenece al usuario).
	ErrNotFound = errors.New("no encontrado")
	// ErrInvalidInput indica que los datos violan una regla de negocio.
	ErrInvalidInput = errors.New("entrada inválida")
	// ErrConflict indica que la operación choca con el estado actual (p. ej. un duplicado).
	ErrConflict = errors.New("conflicto")
)

// ID identifica una entidad persistida. Es un UUID en su forma textual,
// asignado por la base de datos al crear la entidad.
type ID string

// IsZero indica si el ID aún no ha sido asignado.
func (id ID) IsZero() bool {
	return id == ""
}

// Currency es un código de moneda ISO 4217 en mayúsculas (p. ej. "USD").
type Currency string

// ParseCurrency normaliza s y valida que sea un código ISO 4217 de tres letras.
func ParseCurrency(s string) (Currency, error) {
	c := Currency(strings.ToUpper(strings.TrimSpace(s)))
	if !c.Valid() {
		return "", fmt.Errorf("%w: moneda %q no es un código ISO 4217", ErrInvalidInput, s)
	}
	return c, nil
}

// Valid indica si c tiene exactamente tres letras mayúsculas A-Z.
func (c Currency) Valid() bool {
	if len(c) != 3 {
		return false
	}
	for i := range len(c) {
		if c[i] < 'A' || c[i] > 'Z' {
			return false
		}
	}
	return true
}
