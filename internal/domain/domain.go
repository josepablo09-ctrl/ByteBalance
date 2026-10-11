// Package domain contiene los tipos y errores compartidos por todos los módulos
// de dominio de ByteBalance (user, finance, habit). Solo depende de la stdlib.
package domain

import (
	"errors"
	"fmt"
	"strings"
	"uuid"
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

// ID identifica una entidad persistida. Es un UUID asignado por la base de
// datos al crear la entidad; el valor cero indica que aún no se ha persistido.
type ID uuid.UUID

// NewID genera un ID nuevo (UUIDv7, ordenable por fecha de creación).
func NewID() ID {
	return ID(uuid.NewV7())
}

// ParseID convierte la forma textual de un UUID en un ID.
func ParseID(s string) (ID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return ID{}, fmt.Errorf("%w: id %q no es un UUID válido", ErrInvalidInput, s)
	}
	return ID(u), nil
}

// MustParseID es como ParseID pero entra en pánico si s no es válido.
// Solo debe usarse con valores constantes (p. ej. en tests).
func MustParseID(s string) ID {
	return ID(uuid.MustParse(s))
}

// IsZero indica si el ID aún no ha sido asignado.
func (id ID) IsZero() bool {
	return id == ID{}
}

// String devuelve el UUID en su forma canónica (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx).
func (id ID) String() string {
	return uuid.UUID(id).String()
}

// MarshalText implementa encoding.TextMarshaler (el ID se serializa como texto en JSON).
func (id ID) MarshalText() ([]byte, error) {
	return uuid.UUID(id).MarshalText()
}

// UnmarshalText implementa encoding.TextUnmarshaler.
func (id *ID) UnmarshalText(b []byte) error {
	var u uuid.UUID
	if err := u.UnmarshalText(b); err != nil {
		return fmt.Errorf("%w: id no es un UUID válido", ErrInvalidInput)
	}
	*id = ID(u)
	return nil
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
