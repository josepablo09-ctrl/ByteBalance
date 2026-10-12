package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/finance"
)

// Límites de paginación del listado de transacciones.
const (
	defaultTransactionLimit = 50
	maxTransactionLimit     = 200
)

// FinanceService implementa los casos de uso de finanzas personales:
// categorías, registro de movimientos y consulta de balance.
type FinanceService struct {
	categories   finance.CategoryRepository
	transactions finance.TransactionRepository
	users        userFinder
	now          func() time.Time
}

// NewFinanceService crea un FinanceService. Si now es nil se usa time.Now.
func NewFinanceService(
	categories finance.CategoryRepository,
	transactions finance.TransactionRepository,
	users userFinder,
	now func() time.Time,
) *FinanceService {
	return &FinanceService{categories: categories, transactions: transactions, users: users, now: clockOrDefault(now)}
}

// CreateCategory crea una categoría de ingresos o gastos para el usuario.
func (s *FinanceService) CreateCategory(ctx context.Context, userID domain.ID, name string, t finance.EntryType) (*finance.Category, error) {
	c, err := finance.NewCategory(userID, name, t)
	if err != nil {
		return nil, fmt.Errorf("crear categoría: %w", err)
	}
	if err := s.categories.Create(ctx, c); err != nil {
		return nil, fmt.Errorf("crear categoría: %w", err)
	}
	return c, nil
}

// ListCategories devuelve las categorías activas del usuario.
func (s *FinanceService) ListCategories(ctx context.Context, userID domain.ID) ([]finance.Category, error) {
	categories, err := s.categories.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listar categorías: %w", err)
	}
	return categories, nil
}

// RecordTransactionInput son los datos de un ingreso o gasto.
type RecordTransactionInput struct {
	Type finance.EntryType
	// AmountCents es el monto en centavos; debe ser mayor que cero.
	AmountCents int64
	// Currency vacía equivale a la moneda por defecto del usuario.
	Currency string
	// CategoryID es opcional; la categoría debe ser del usuario y del mismo tipo.
	CategoryID  *domain.ID
	Description string
	// OccurredAt en su valor cero equivale al momento actual.
	OccurredAt time.Time
}

// RecordTransaction registra un ingreso o gasto del usuario.
func (s *FinanceService) RecordTransaction(ctx context.Context, userID domain.ID, in RecordTransactionInput) (*finance.Transaction, error) {
	currency, err := s.resolveCurrency(ctx, userID, in.Currency)
	if err != nil {
		return nil, fmt.Errorf("registrar transacción: %w", err)
	}

	var category *finance.Category
	if in.CategoryID != nil {
		// GetByID está acotado al usuario: una categoría ajena se reporta como inexistente.
		category, err = s.categories.GetByID(ctx, userID, *in.CategoryID)
		if err != nil {
			return nil, fmt.Errorf("registrar transacción: categoría: %w", err)
		}
	}

	occurredAt := in.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = s.now()
	}
	t, err := finance.NewTransaction(finance.TransactionParams{
		UserID:      userID,
		Type:        in.Type,
		Amount:      finance.Money{Cents: in.AmountCents, Currency: currency},
		Description: in.Description,
		OccurredAt:  occurredAt,
		Category:    category,
	})
	if err != nil {
		return nil, fmt.Errorf("registrar transacción: %w", err)
	}

	if err := s.transactions.Create(ctx, t); err != nil {
		return nil, fmt.Errorf("registrar transacción: %w", err)
	}
	return t, nil
}

// resolveCurrency valida la moneda indicada o, si está vacía, usa la del usuario.
func (s *FinanceService) resolveCurrency(ctx context.Context, userID domain.ID, raw string) (domain.Currency, error) {
	if strings.TrimSpace(raw) != "" {
		return domain.ParseCurrency(raw)
	}
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("obtener moneda por defecto: %w", err)
	}
	return u.DefaultCurrency, nil
}

// ListTransactions devuelve las transacciones del usuario que cumplen f. Sin
// Limit se devuelven 50, y nunca más de 200.
func (s *FinanceService) ListTransactions(ctx context.Context, userID domain.ID, f finance.TransactionFilter) ([]finance.Transaction, error) {
	if err := validateRange(f.From, f.To); err != nil {
		return nil, fmt.Errorf("listar transacciones: %w", err)
	}
	if f.Offset < 0 {
		return nil, fmt.Errorf("listar transacciones: %w: el offset no puede ser negativo", domain.ErrInvalidInput)
	}
	if f.Type != "" && !f.Type.Valid() {
		return nil, fmt.Errorf("listar transacciones: %w: tipo %q inválido", domain.ErrInvalidInput, f.Type)
	}
	if f.Limit <= 0 {
		f.Limit = defaultTransactionLimit
	}
	f.Limit = min(f.Limit, maxTransactionLimit)

	transactions, err := s.transactions.List(ctx, userID, f)
	if err != nil {
		return nil, fmt.Errorf("listar transacciones: %w", err)
	}
	return transactions, nil
}

// Balance devuelve los ingresos, gastos y neto del usuario por moneda en el
// intervalo [from, to). Un extremo en su valor cero no limita.
func (s *FinanceService) Balance(ctx context.Context, userID domain.ID, from, to time.Time) ([]finance.Balance, error) {
	if err := validateRange(from, to); err != nil {
		return nil, fmt.Errorf("calcular balance: %w", err)
	}
	// Sin Limit: el balance debe considerar todas las transacciones del periodo.
	transactions, err := s.transactions.List(ctx, userID, finance.TransactionFilter{From: from, To: to})
	if err != nil {
		return nil, fmt.Errorf("calcular balance: %w", err)
	}
	return finance.Summarize(transactions), nil
}

// validateRange comprueba que el intervalo [from, to) no esté vacío ni invertido.
func validateRange(from, to time.Time) error {
	if !from.IsZero() && !to.IsZero() && !from.Before(to) {
		return fmt.Errorf("%w: la fecha inicial debe ser anterior a la final", domain.ErrInvalidInput)
	}
	return nil
}
