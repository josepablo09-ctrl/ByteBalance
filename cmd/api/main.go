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

	"github.com/jackc/pgx/v5/pgxpool"

	"bytebalance/internal/config"
	httpdelivery "bytebalance/internal/delivery/http"
	"bytebalance/internal/domain/finance"
	"bytebalance/internal/domain/habit"
	"bytebalance/internal/domain/user"
	"bytebalance/internal/repository/postgres"
	"bytebalance/internal/security"
	"bytebalance/internal/usecase"
)

const (
	dbConnectTimeout = 10 * time.Second
	shutdownTimeout  = 10 * time.Second
	accessTokenTTL   = 24 * time.Hour
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

// newHandler construye los casos de uso sobre los repositorios y los expone
// mediante el router HTTP.
func newHandler(cfg config.Config, pool *pgxpool.Pool) (http.Handler, error) {
	repos := newRepositories(pool)

	hasher, err := security.NewBcryptHasher(security.DefaultBcryptCost)
	if err != nil {
		return nil, fmt.Errorf("crear hasher de contraseñas: %w", err)
	}
	tokens, err := security.NewJWT([]byte(cfg.JWTSecret), accessTokenTTL, time.Now)
	if err != nil {
		return nil, fmt.Errorf("crear emisor de tokens: %w", err)
	}

	return httpdelivery.NewRouter(httpdelivery.Services{
		Users:   usecase.NewUserService(repos.users, hasher),
		Auth:    usecase.NewAuthService(repos.users, hasher, tokens),
		Finance: usecase.NewFinanceService(repos.categories, repos.transactions, repos.users, time.Now),
		Habits:  usecase.NewHabitService(repos.habits, repos.habitLogs, repos.users, time.Now),
		Tokens:  tokens,
		DB:      pool,
	}, httpdelivery.Options{
		TrustedProxyHops: cfg.TrustedProxyHops,
	}), nil
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

	handler, err := newHandler(cfg, pool)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           handler,
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
