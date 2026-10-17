package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/finance"
	"bytebalance/internal/domain/habit"
	"bytebalance/internal/domain/insight"
)

// fakeAI simula la IA: registra el resumen recibido y devuelve una respuesta fija.
type fakeAI struct {
	diagnosis insight.Diagnosis
	err       error
	calls     int
	got       insight.Snapshot
}

func (f *fakeAI) Diagnose(_ context.Context, s insight.Snapshot) (insight.Diagnosis, error) {
	f.calls++
	f.got = s
	return f.diagnosis, f.err
}

// insightFixture arma un InsightService con un usuario en Costa Rica (UTC-6).
// Ahora son las 03:00 UTC del 16/10, es decir, las 21:00 del 15/10 para él.
type insightFixture struct {
	userID       domain.ID
	categories   *fakeCategoryRepo
	transactions *fakeTransactionRepo
	habits       *fakeHabitRepo
	logs         *fakeLogRepo
	ai           *fakeAI
	svc          *InsightService
}

var insightNow = time.Date(2026, 10, 16, 3, 0, 0, 0, time.UTC)

func newInsightFixture(t *testing.T) *insightFixture {
	t.Helper()
	u := existingUser(t, "ana@example.com", "America/Costa_Rica")
	goal := int64(100_000_00)
	u.MonthlySavingsGoalCents = &goal
	f := &insightFixture{
		userID:       u.ID,
		categories:   newFakeCategoryRepo(),
		transactions: &fakeTransactionRepo{},
		habits:       newFakeHabitRepo(),
		logs:         newFakeLogRepo(),
		ai:           &fakeAI{diagnosis: insight.Diagnosis{Headline: "**Vas bien**", Summary: "Resumen."}},
	}
	f.svc = NewInsightService(newFakeUserRepo(u), f.categories, f.transactions, f.habits, f.logs, f.ai,
		func() time.Time { return insightNow })
	return f
}

func (f *insightFixture) addTransaction(typ finance.EntryType, cents int64, at time.Time, category *domain.ID) {
	f.transactions.stored = append(f.transactions.stored, finance.Transaction{
		ID: domain.NewID(), UserID: f.userID, Type: typ, CategoryID: category,
		Amount: finance.Money{Cents: cents, Currency: "CRC"}, Description: "Cena con Juan", OccurredAt: at,
	})
}

func TestInsightServiceDiagnose(t *testing.T) {
	ctx := context.Background()
	f := newInsightFixture(t)
	utc := func(m time.Month, d, h int) time.Time { return time.Date(2026, m, d, h, 0, 0, 0, time.UTC) }

	food := finance.Category{ID: domain.NewID(), UserID: f.userID, Name: "Comida", Type: finance.Expense}
	f.categories.byID[food.ID] = food

	// El periodo actual va del 16/09 al 15/10 en Costa Rica: desde las 06:00 UTC del 16/09.
	f.addTransaction(finance.Income, 500_000_00, utc(10, 1, 15), nil)
	f.addTransaction(finance.Expense, 80_000_00, utc(10, 16, 2), &food.ID) // 15/10 a las 20:00 locales
	f.addTransaction(finance.Expense, 20_000_00, utc(9, 16, 7), &food.ID)  // primer día del periodo
	f.addTransaction(finance.Expense, 70_000_00, utc(9, 16, 5), &food.ID)  // 15/09 local: periodo anterior
	f.addTransaction(finance.Income, 1_00, utc(8, 16, 12), nil)            // fuera de ambos periodos
	f.addTransaction(finance.Expense, 9_00, utc(10, 16, 7), nil)           // 16/10 local: aún no ocurre

	meditate := habit.Habit{ID: domain.NewID(), UserID: f.userID, Name: "Meditar", Frequency: habit.Daily, TargetCount: 1, IsActive: true,
		CreatedAt: utc(10, 13, 15)} // creado el 13/10
	paused := habit.Habit{ID: domain.NewID(), UserID: f.userID, Name: "Pausado", Frequency: habit.Daily, TargetCount: 1}
	foreign := habit.Habit{ID: domain.NewID(), UserID: domain.NewID(), Name: "Ajeno", Frequency: habit.Daily, TargetCount: 1, IsActive: true}
	for _, h := range []habit.Habit{meditate, paused, foreign} {
		f.habits.byID[h.ID] = h
	}
	for _, d := range []int{13, 14} {
		if err := f.logs.Upsert(ctx, &habit.Log{HabitID: meditate.ID, LoggedOn: time.Date(2026, 10, d, 0, 0, 0, 0, time.UTC), Count: 1}); err != nil {
			t.Fatal(err)
		}
	}

	report, err := f.svc.Diagnose(ctx, f.userID)
	if err != nil {
		t.Fatalf("Diagnose() error inesperado: %v", err)
	}

	if report.Headline != "Vas bien" || report.Summary != "Resumen." {
		t.Errorf("el diagnóstico no se saneó: %+v", report.Diagnosis)
	}
	if !report.GeneratedAt.Equal(insightNow) {
		t.Errorf("GeneratedAt = %v", report.GeneratedAt)
	}
	wantFrom, wantTo := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
	if !report.From.Equal(wantFrom) || !report.To.Equal(wantTo) {
		t.Errorf("periodo = %v – %v, se esperaba %v – %v", report.From, report.To, wantFrom, wantTo)
	}
	if f.transactions.lastFilter.Limit != maxAnalyzedTransactions {
		t.Errorf("Limit = %d, se esperaba %d", f.transactions.lastFilter.Limit, maxAnalyzedTransactions)
	}

	got := f.ai.got
	if got.SavingsGoal == nil || *got.SavingsGoal != (finance.Money{Cents: 100_000_00, Currency: "CRC"}) {
		t.Errorf("SavingsGoal = %v, se esperaba la meta del usuario en su moneda", got.SavingsGoal)
	}
	if len(got.Finances) != 1 {
		t.Fatalf("Finances = %+v, se esperaba solo CRC", got.Finances)
	}
	crc := got.Finances[0]
	if crc.IncomeCents != 500_000_00 || crc.ExpenseCents != 100_000_00 || crc.PreviousExpenseCents != 70_000_00 || crc.Transactions != 3 {
		t.Errorf("resumen CRC = %+v", crc)
	}
	if len(crc.TopExpenses) != 1 || crc.TopExpenses[0] != (insight.CategoryAmount{Name: "Comida", Cents: 100_000_00}) {
		t.Errorf("TopExpenses = %+v", crc.TopExpenses)
	}
	wantHabit := insight.HabitSummary{Name: "Meditar", Frequency: habit.Daily, TargetCount: 1, PeriodsMet: 2, PeriodsTotal: 2, CurrentStreak: 2}
	if len(got.Habits) != 1 || got.Habits[0] != wantHabit {
		t.Errorf("Habits = %+v, se esperaba solo %+v", got.Habits, wantHabit)
	}
}

func TestInsightServiceDiagnoseErrors(t *testing.T) {
	errDB := errors.New("base caída")
	errGoogle := errors.New("gemini: HTTP 503")
	withExpense := func(f *insightFixture) {
		f.addTransaction(finance.Expense, 1_000_00, insightNow.Add(-time.Hour), nil)
	}

	tests := []struct {
		name      string
		setup     func(f *insightFixture)
		userID    func(f *insightFixture) domain.ID
		wantErr   error
		wantCalls int
	}{
		{name: "sin datos no llama a la IA", setup: func(*insightFixture) {}, wantErr: insight.ErrNotEnoughData},
		{
			name:    "usuario inexistente",
			setup:   withExpense,
			userID:  func(*insightFixture) domain.ID { return domain.NewID() },
			wantErr: domain.ErrNotFound,
		},
		{
			name:    "error al leer transacciones",
			setup:   func(f *insightFixture) { f.transactions.listErr = errDB },
			wantErr: errDB,
		},
		{
			name: "la IA falla", wantCalls: 1, wantErr: ErrAIUnavailable,
			setup: func(f *insightFixture) { withExpense(f); f.ai.err = errGoogle },
		},
		{
			name: "la IA responde vacío", wantCalls: 1, wantErr: insight.ErrInvalidDiagnosis,
			setup: func(f *insightFixture) { withExpense(f); f.ai.diagnosis = insight.Diagnosis{Headline: "  "} },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newInsightFixture(t)
			tt.setup(f)
			id := f.userID
			if tt.userID != nil {
				id = tt.userID(f)
			}
			_, err := f.svc.Diagnose(context.Background(), id)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Diagnose() error = %v, se esperaba %v", err, tt.wantErr)
			}
			if f.ai.calls != tt.wantCalls {
				t.Errorf("llamadas a la IA = %d, se esperaban %d", f.ai.calls, tt.wantCalls)
			}
		})
	}
}

func TestInsightServiceDiagnoseInvalidResponseIsUnavailable(t *testing.T) {
	f := newInsightFixture(t)
	f.addTransaction(finance.Income, 1_000_00, insightNow.Add(-time.Hour), nil)
	f.ai.diagnosis = insight.Diagnosis{}
	if _, err := f.svc.Diagnose(context.Background(), f.userID); !errors.Is(err, ErrAIUnavailable) {
		t.Fatalf("Diagnose() error = %v, una respuesta inválida debe reportarse como ErrAIUnavailable", err)
	}
}
