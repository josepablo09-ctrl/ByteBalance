package postgres

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/user"
)

// testTx abre una transacción contra DATABASE_URL que se revierte al terminar
// el test, de modo que los tests de integración no dejan datos en la base.
// Si DATABASE_URL no está definida, el test se omite.
func testTx(t *testing.T) pgx.Tx {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL no definida: se omiten los tests de integración con PostgreSQL")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("conectar con postgres: %v", err)
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("iniciar transacción: %v", err)
	}
	t.Cleanup(func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("revertir transacción: %v", err)
		}
		if err := conn.Close(ctx); err != nil {
			t.Errorf("cerrar conexión: %v", err)
		}
	})
	return tx
}

// expectErrInSavepoint ejecuta fn dentro de un savepoint y comprueba que falle
// con want. El savepoint evita que el error aborte la transacción del test.
func expectErrInSavepoint(t *testing.T, tx pgx.Tx, want error, fn func(db DBTX) error) {
	t.Helper()
	ctx := context.Background()
	sp, err := tx.Begin(ctx)
	if err != nil {
		t.Fatalf("crear savepoint: %v", err)
	}
	if err := fn(sp); !errors.Is(err, want) {
		t.Errorf("error = %v, se esperaba %v", err, want)
	}
	if err := sp.Rollback(ctx); err != nil {
		t.Fatalf("revertir savepoint: %v", err)
	}
}

// createTestUser persiste un usuario válido con un email único.
func createTestUser(t *testing.T, db DBTX) *user.User {
	t.Helper()
	u, err := user.New(user.Params{
		Email:           domain.NewID().String() + "@test.bytebalance.dev",
		PasswordHash:    "hash",
		FullName:        "Usuario de prueba",
		DefaultCurrency: "USD",
	})
	if err != nil {
		t.Fatalf("user.New: %v", err)
	}
	if err := NewUserRepository(db).Create(context.Background(), u); err != nil {
		t.Fatalf("crear usuario de prueba: %v", err)
	}
	return u
}
