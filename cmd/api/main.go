// Package main es el entrypoint del backend de ByteBalance.
// Solo contiene el wiring de dependencias; la lógica de negocio vive en internal/.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	// Incluye la base de zonas horarias en el binario para que time.LoadLocation
	// funcione en imágenes mínimas (distroless/scratch) sin tzdata del sistema.
	_ "time/tzdata"

	"bytebalance/internal/config"
	httpdelivery "bytebalance/internal/delivery/http"
	"bytebalance/internal/domain/finance"
	"bytebalance/internal/domain/habit"
	"bytebalance/internal/domain/user"
	"bytebalance/internal/repository/postgres"
)

const (
	dbConnectTimeout = 10 * time.Second
	shutdownTimeout  = 10 * time.Second
)

// repositories agrupa las implementaciones de persistencia que se inyectarán
// en los casos de uso. Los campos se tipan con las interfaces del dominio.
type repositories struct {
	users        user.Repository
	categories   finance.CategoryRepository
	transactions finance.TransactionRepository
	habits       habit.Repository
	habitLogs    habit.LogRepository
}

func newRepositories(db postgres.DBTX) repositories {
	return repositories{
		users:        postgres.NewUserRepository(db),
		categories:   postgres.NewCategoryRepository(db),
		transactions: postgres.NewTransactionRepository(db),
		habits:       postgres.NewHabitRepository(db),
		habitLogs:    postgres.NewHabitLogRepository(db),
	}
}

// app reúne las dependencias construidas en el arranque.
type app struct {
	db httpdelivery.Pinger
	// repos se inyectará en los casos de uso a medida que se implementen.
	repos repositories
}

// routes registra los endpoints HTTP de la API.
func (a *app) routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", httpdelivery.Health(a.db))
	return mux
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if err := run(); err != nil {
		slog.Error("la api terminó con error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("cargar configuración: %w", err)
	}

	connectCtx, cancel := context.WithTimeout(ctx, dbConnectTimeout)
	pool, err := postgres.NewPool(connectCtx, cfg.DatabaseURL)
	cancel()
	if err != nil {
		return err
	}
	defer pool.Close()

	a := &app{db: pool, repos: newRepositories(pool)}

	srv := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           a.routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		slog.Info("bytebalance api escuchando", "addr", srv.Addr, "env", cfg.Env)
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		return fmt.Errorf("servidor http: %w", err)
	case <-ctx.Done():
	}

	slog.Info("apagando bytebalance api")
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancelShutdown()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("apagar servidor http: %w", err)
	}
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("servidor http: %w", err)
	}
	return nil
}
