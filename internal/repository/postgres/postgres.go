// Package postgres implementa los repositorios del dominio sobre PostgreSQL con pgx.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"bytebalance/internal/domain"
)

// DBTX es el subconjunto de operaciones que usan los repositorios. Lo cumplen
// *pgxpool.Pool, *pgx.Conn y pgx.Tx, de modo que un caso de uso puede ejecutar
// varios repositorios dentro de una misma transacción.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// NewPool abre un pool de conexiones a PostgreSQL y verifica que la base responda.
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		// pgx oculta la contraseña en el mensaje de error.
		return nil, fmt.Errorf("parsear configuración de postgres: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("crear pool de postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("conectar con postgres: %w", err)
	}
	return pool, nil
}

// Códigos SQLSTATE que se traducen a errores de dominio.
const (
	uniqueViolation     = "23505"
	foreignKeyViolation = "23503"
	checkViolation      = "23514"
	stringTooLong       = "22001"
)

// mapError traduce los errores de pgx/PostgreSQL a errores de dominio,
// conservando el error original en la cadena para diagnóstico.
func mapError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case uniqueViolation:
			return fmt.Errorf("%w: %w", domain.ErrConflict, err)
		case foreignKeyViolation:
			return fmt.Errorf("%w: referencia inexistente: %w", domain.ErrNotFound, err)
		case checkViolation, stringTooLong:
			return fmt.Errorf("%w: %w", domain.ErrInvalidInput, err)
		}
	}
	return err
}

// requireAffected devuelve domain.ErrNotFound si la sentencia no modificó filas.
func requireAffected(tag pgconn.CommandTag) error {
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// uuidArg convierte un ID de dominio al tipo que pgx codifica como UUID.
func uuidArg(id domain.ID) [16]byte {
	return [16]byte(id)
}

// uuidDst permite escanear una columna UUID directamente sobre un ID de dominio.
func uuidDst(id *domain.ID) *[16]byte {
	return (*[16]byte)(id)
}

// nullableUUIDArg convierte un ID opcional en un argumento UUID o NULL.
func nullableUUIDArg(id *domain.ID) any {
	if id == nil {
		return nil
	}
	return uuidArg(*id)
}

// idFromNullable convierte una columna UUID nullable en un ID opcional.
func idFromNullable(u pgtype.UUID) *domain.ID {
	if !u.Valid {
		return nil
	}
	id := domain.ID(u.Bytes)
	return &id
}
