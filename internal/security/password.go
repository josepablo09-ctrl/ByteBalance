// Package security implementa las primitivas criptográficas de la aplicación:
// hash de contraseñas y tokens de acceso.
package security

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// DefaultBcryptCost es el costo de bcrypt usado en producción (2^12 rondas).
const DefaultBcryptCost = 12

// BcryptHasher genera y verifica hashes de contraseñas con bcrypt.
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher crea un BcryptHasher con el costo indicado, que debe estar
// entre bcrypt.MinCost y bcrypt.MaxCost.
func NewBcryptHasher(cost int) (*BcryptHasher, error) {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		return nil, fmt.Errorf("costo de bcrypt %d fuera de rango [%d, %d]", cost, bcrypt.MinCost, bcrypt.MaxCost)
	}
	return &BcryptHasher{cost: cost}, nil
}

// Hash devuelve el hash bcrypt de password (incluye la sal). bcrypt solo admite
// contraseñas de hasta 72 bytes.
func (h *BcryptHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("generar hash bcrypt: %w", err)
	}
	return string(hash), nil
}

// Verify indica si password corresponde a hash. Devuelve error solo si el hash
// es inválido; una contraseña incorrecta devuelve false sin error.
func (h *BcryptHasher) Verify(hash, password string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
		return false, nil
	default:
		return false, fmt.Errorf("verificar hash bcrypt: %w", err)
	}
}
