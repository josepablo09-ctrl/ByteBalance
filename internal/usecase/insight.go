package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/finance"
	"bytebalance/internal/domain/habit"
	"bytebalance/internal/domain/insight"
)

// maxAnalyzedTransactions acota las transacciones que se leen para un análisis
// (periodo actual y anterior). Por encima, los totales serían aproximados.
const maxAnalyzedTransactions = 5000

// ErrAIUnavailable indica que el servicio de IA no pudo generar el diagnóstico
// (caído, saturado, sin cuota o con una respuesta inválida).
var ErrAIUnavailable = errors.New("el servicio de IA no está disponible")

// DiagnosisGenerator genera un diagnóstico a partir de un resumen anónimo.
type DiagnosisGenerator interface {
	Diagnose(ctx context.Context, s insight.Snapshot) (insight.Diagnosis, error)
}

// InsightReport es un diagnóstico listo para mostrar y el periodo que analiza.
type InsightReport struct {
	insight.Diagnosis
	// From y To son las fechas (inclusive) del periodo analizado.
	From        time.Time
	To          time.Time
	GeneratedAt time.Time
}

// InsightService genera diagnósticos personalizados con IA a partir de las
// finanzas y los hábitos del usuario.
type InsightService struct {
	users        userFinder
	categories   finance.CategoryRepository
	transactions finance.TransactionRepository
	habits       habit.Repository
	logs         habit.LogRepository
	ai           DiagnosisGenerator
	now          func() time.Time
}

// NewInsightService crea un InsightService. Si now es nil se usa time.Now.
func NewInsightService(
	users userFinder,
	categories finance.CategoryRepository,
	transactions finance.TransactionRepository,
	habits habit.Repository,
	logs habit.LogRepository,
	ai DiagnosisGenerator,
	now func() time.Time,
) *InsightService {
	return &InsightService{
		users: users, categories: categories, transactions: transactions,
		habits: habits, logs: logs, ai: ai, now: clockOrDefault(now),
	}
}

// Diagnose analiza los últimos insight.WindowDays días del usuario, en su zona
// horaria, y devuelve el diagnóstico saneado. Devuelve insight.ErrNotEnoughData
// si no hay transacciones ni hábitos, sin llamar a la IA, y ErrAIUnavailable si
// la IA falla.
func (s *InsightService) Diagnose(ctx context.Context, userID domain.ID) (InsightReport, error) {
	snapshot, err := s.snapshot(ctx, userID)
	if err != nil {
		return InsightReport{}, fmt.Errorf("generar diagnóstico: %w", err)
	}
	if snapshot.IsEmpty() {
		return InsightReport{}, fmt.Errorf("generar diagnóstico: %w", insight.ErrNotEnoughData)
	}

	raw, err := s.ai.Diagnose(ctx, snapshot)
	if err != nil {
		// El detalle (código de Google, cuota…) solo va al log; no contiene datos del usuario.
		slog.WarnContext(ctx, "la IA no generó el diagnóstico", "error", err)
		return InsightReport{}, fmt.Errorf("generar diagnóstico: %w: %w", ErrAIUnavailable, err)
	}
	diagnosis, err := raw.Sanitized()
	if err != nil {
		slog.WarnContext(ctx, "la IA devolvió un diagnóstico inválido", "error", err)
		return InsightReport{}, fmt.Errorf("generar diagnóstico: %w: %w", ErrAIUnavailable, err)
	}
	return InsightReport{Diagnosis: diagnosis, From: snapshot.From, To: snapshot.To, GeneratedAt: s.now()}, nil
}

// snapshot reúne y agrega los datos del usuario. Solo lee lo necesario y
// descarta descripciones, notas e identificadores.
func (s *InsightService) snapshot(ctx context.Context, userID domain.ID) (insight.Snapshot, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return insight.Snapshot{}, fmt.Errorf("obtener usuario: %w", err)
	}
	loc, err := u.Location()
	if err != nil {
		return insight.Snapshot{}, err
	}

	now := s.now().In(loc)
	today := habit.DateOf(now)
	from := today.AddDate(0, 0, -(insight.WindowDays - 1))
	// Límites de los periodos como instantes en la zona del usuario.
	currentStart := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, loc)
	previousStart := currentStart.AddDate(0, 0, -insight.WindowDays)
	tomorrow := time.Date(today.Year(), today.Month(), today.Day()+1, 0, 0, 0, 0, loc)

	finances, err := s.summarizeFinances(ctx, userID, previousStart, currentStart, tomorrow)
	if err != nil {
		return insight.Snapshot{}, err
	}
	habits, err := s.summarizeHabits(ctx, userID, loc, from, today)
	if err != nil {
		return insight.Snapshot{}, err
	}
	return insight.Snapshot{From: from, To: today, Finances: finances, Habits: habits}, nil
}

func (s *InsightService) summarizeFinances(ctx context.Context, userID domain.ID, previousStart, currentStart, end time.Time) ([]insight.CurrencySummary, error) {
	transactions, err := s.transactions.List(ctx, userID, finance.TransactionFilter{From: previousStart, To: end, Limit: maxAnalyzedTransactions})
	if err != nil {
		return nil, fmt.Errorf("listar transacciones: %w", err)
	}
	var current, previous []finance.Transaction
	for _, t := range transactions {
		if t.OccurredAt.Before(currentStart) {
			previous = append(previous, t)
		} else {
			current = append(current, t)
		}
	}

	categories, err := s.categories.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listar categorías: %w", err)
	}
	names := make(map[domain.ID]string, len(categories))
	for _, c := range categories {
		names[c.ID] = c.Name
	}
	return insight.SummarizeFinances(current, previous, names), nil
}

func (s *InsightService) summarizeHabits(ctx context.Context, userID domain.ID, loc *time.Location, from, today time.Time) ([]insight.HabitSummary, error) {
	habits, err := s.habits.ListByUser(ctx, userID, true)
	if err != nil {
		return nil, fmt.Errorf("listar hábitos: %w", err)
	}
	if len(habits) > insight.MaxHabits {
		habits = habits[:insight.MaxHabits]
	}

	summaries := make([]insight.HabitSummary, 0, len(habits))
	for _, h := range habits {
		// Un hábito creado dentro del periodo solo se evalúa desde su creación.
		start := from
		if created := habit.DateOf(h.CreatedAt.In(loc)); created.After(start) {
			start = created
		}
		if start.After(today) {
			start = today
		}
		logs, err := s.logs.ListByHabit(ctx, h.ID, habit.PeriodStart(h.Frequency, start), today)
		if err != nil {
			return nil, fmt.Errorf("listar registros de hábitos: %w", err)
		}
		summaries = append(summaries, insight.SummarizeHabit(h, logs, start, today))
	}
	return summaries, nil
}
