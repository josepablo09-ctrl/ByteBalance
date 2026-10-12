// Package usecase implementa los casos de uso de ByteBalance. Orquesta las
// entidades del dominio y recibe sus dependencias (repositorios, servicios
// externos) como interfaces, nunca como implementaciones concretas.
package usecase

import (
	"context"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/user"
)

// userFinder es la parte de user.Repository que necesitan los casos de uso que
// solo consultan al usuario (p. ej. su moneda o su zona horaria).
type userFinder interface {
	GetByID(ctx context.Context, id domain.ID) (*user.User, error)
}

// clockOrDefault devuelve now, o time.Now si es nil.
func clockOrDefault(now func() time.Time) func() time.Time {
	if now == nil {
		return time.Now
	}
	return now
}
