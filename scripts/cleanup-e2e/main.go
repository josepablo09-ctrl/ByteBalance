// Command cleanup-e2e borra de la base de datos local los usuarios creados por
// las pruebas end-to-end (emails e2e+…@bytebalance.test) junto con todos sus
// datos: categorías, transacciones, hábitos y registros (ON DELETE CASCADE).
//
// Por seguridad solo corre con APP_ENV=development contra un PostgreSQL en
// localhost, y sin -yes solo muestra lo que borraría.
//
// Uso, desde la raíz del repositorio:
//
//	go run ./scripts/cleanup-e2e        # simulación: cuenta, no borra
//	go run ./scripts/cleanup-e2e -yes   # borra
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"

	"bytebalance/internal/config"
	"bytebalance/internal/repository/postgres"
)

// e2eEmailPattern es el patrón LIKE de los emails que usan las pruebas e2e. El
// dominio .test está reservado (RFC 2606): ningún usuario real puede tenerlo.
const e2eEmailPattern = "e2e+%@bytebalance.test"

const timeout = 30 * time.Second

// localHosts son los únicos hosts de base de datos en los que se permite borrar.
var localHosts = []string{"localhost", "127.0.0.1", "::1"}

// usersQuery selecciona los usuarios de prueba, incluidos los dados de baja.
const usersQuery = `SELECT id FROM users WHERE lower(email) LIKE $1`

// summary cuenta las filas que se borrarían.
type summary struct {
	users, categories, transactions, habits, habitLogs int64
}

func (s summary) String() string {
	return fmt.Sprintf("%d usuarios, %d categorías, %d transacciones, %d hábitos, %d registros de hábitos",
		s.users, s.categories, s.transactions, s.habits, s.habitLogs)
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		slog.Error("cleanup-e2e terminó con error", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, out io.Writer) error {
	flags := flag.NewFlagSet("cleanup-e2e", flag.ContinueOnError)
	confirm := flags.Bool("yes", false, "borra de verdad; sin esta opción solo simula")
	if err := flags.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("cargar configuración: %w", err)
	}
	if err := checkSafe(cfg); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("iniciar transacción: %w", err)
	}
	// Tras un Commit exitoso, Rollback no hace nada; si algo falla, deshace todo.
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			slog.Error("revertir transacción", "error", err)
		}
	}()

	found, err := count(ctx, tx)
	if err != nil {
		return err
	}
	if found.users == 0 {
		fmt.Fprintln(out, "No hay usuarios de prueba e2e: la base ya está limpia.")
		return nil
	}
	if !*confirm {
		fmt.Fprintf(out, "Simulación: se borrarían %s.\nVuelve a ejecutar con -yes para borrarlos.\n", found)
		return nil
	}

	tag, err := tx.Exec(ctx, `DELETE FROM users WHERE lower(email) LIKE $1`, e2eEmailPattern)
	if err != nil {
		return fmt.Errorf("borrar usuarios de prueba: %w", err)
	}
	if tag.RowsAffected() != found.users {
		return fmt.Errorf("se esperaba borrar %d usuarios y se borraron %d: no se guardó ningún cambio", found.users, tag.RowsAffected())
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("confirmar borrado: %w", err)
	}
	fmt.Fprintf(out, "Borrados %s.\n", found)
	return nil
}

// checkSafe impide correr el borrado fuera del entorno local de desarrollo.
func checkSafe(cfg config.Config) error {
	if !cfg.IsDevelopment() {
		return fmt.Errorf("cleanup-e2e solo corre con APP_ENV=development (actual: %s)", cfg.Env)
	}
	u, err := url.Parse(cfg.DatabaseURL)
	if err != nil {
		// No se incluye la URL: contiene la contraseña.
		return errors.New("DATABASE_URL no es una URL válida")
	}
	if !slices.Contains(localHosts, u.Hostname()) {
		return fmt.Errorf("cleanup-e2e solo corre contra un PostgreSQL local; DATABASE_URL apunta a %q", u.Hostname())
	}
	return nil
}

// count cuenta las filas de los usuarios de prueba y de todo lo que se borra en cascada.
func count(ctx context.Context, tx pgx.Tx) (summary, error) {
	const q = `WITH target AS (` + usersQuery + `)
		SELECT
			(SELECT count(*) FROM target),
			(SELECT count(*) FROM categories WHERE user_id IN (SELECT id FROM target)),
			(SELECT count(*) FROM transactions WHERE user_id IN (SELECT id FROM target)),
			(SELECT count(*) FROM habits WHERE user_id IN (SELECT id FROM target)),
			(SELECT count(*) FROM habit_logs
				WHERE habit_id IN (SELECT h.id FROM habits h WHERE h.user_id IN (SELECT id FROM target)))`
	var s summary
	err := tx.QueryRow(ctx, q, e2eEmailPattern).
		Scan(&s.users, &s.categories, &s.transactions, &s.habits, &s.habitLogs)
	if err != nil {
		return summary{}, fmt.Errorf("contar datos de prueba: %w", err)
	}
	return s, nil
}
