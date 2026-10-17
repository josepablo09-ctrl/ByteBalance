package insight

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/finance"
	"bytebalance/internal/domain/habit"
)

func day(m time.Month, d int) time.Time {
	return time.Date(2026, m, d, 0, 0, 0, 0, time.UTC)
}

func TestSummarizeFinances(t *testing.T) {
	food, transport, deleted := domain.NewID(), domain.NewID(), domain.NewID()
	names := map[domain.ID]string{food: "  Comida   y  súper ", transport: "Transporte"}
	tx := func(typ finance.EntryType, cents int64, currency domain.Currency, category *domain.ID) finance.Transaction {
		return finance.Transaction{Type: typ, Amount: finance.Money{Cents: cents, Currency: currency}, CategoryID: category}
	}

	current := []finance.Transaction{
		tx(finance.Income, 600_000_00, "CRC", nil),
		tx(finance.Expense, 120_000_00, "CRC", &food),
		tx(finance.Expense, 30_000_00, "CRC", &food),
		tx(finance.Expense, 40_000_00, "CRC", &transport),
		tx(finance.Expense, 5_000_00, "CRC", nil),
		tx(finance.Expense, 1_000_00, "CRC", &deleted), // categoría eliminada
		tx(finance.Expense, 20_00, "USD", nil),
	}
	previous := []finance.Transaction{
		tx(finance.Income, 500_000_00, "CRC", nil),
		tx(finance.Expense, 210_000_00, "CRC", &food),
		tx(finance.Income, 300_00, "EUR", nil), // moneda que dejó de usarse
	}

	got := SummarizeFinances(current, previous, names)

	want := []CurrencySummary{
		{
			Currency: "CRC", IncomeCents: 600_000_00, ExpenseCents: 196_000_00,
			PreviousIncomeCents: 500_000_00, PreviousExpenseCents: 210_000_00, Transactions: 6,
			TopExpenses: []CategoryAmount{
				{Name: "Comida y súper", Cents: 150_000_00},
				{Name: "Transporte", Cents: 40_000_00},
				{Name: UncategorizedName, Cents: 6_000_00},
			},
		},
		{Currency: "USD", ExpenseCents: 20_00, Transactions: 1, TopExpenses: []CategoryAmount{{Name: UncategorizedName, Cents: 20_00}}},
		{Currency: "EUR", PreviousIncomeCents: 300_00},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("SummarizeFinances() =\n%+v\nse esperaba\n%+v", got, want)
	}
}

func TestSummarizeFinancesLimitsTopExpenses(t *testing.T) {
	names := map[domain.ID]string{}
	var current []finance.Transaction
	for i := range 8 {
		id := domain.NewID()
		names[id] = strings.Repeat("x", i+1)
		current = append(current, finance.Transaction{Type: finance.Expense, Amount: finance.Money{Cents: int64(100 * (i + 1)), Currency: "USD"}, CategoryID: &id})
	}
	got := SummarizeFinances(current, nil, names)
	if len(got) != 1 || len(got[0].TopExpenses) != maxTopExpenses || got[0].TopExpenses[0].Cents != 800 {
		t.Fatalf("TopExpenses = %+v, se esperaban las %d mayores en orden", got[0].TopExpenses, maxTopExpenses)
	}
}

func TestSummarizeHabit(t *testing.T) {
	logOn := func(d time.Time, count int) habit.Log { return habit.Log{LoggedOn: d, Count: count} }
	// Viernes 16 de octubre de 2026.
	today := day(10, 16)

	tests := []struct {
		name  string
		habit habit.Habit
		logs  []habit.Log
		start time.Time
		want  HabitSummary
	}{
		{
			name:  "diario: racha y día en curso pendiente",
			habit: habit.Habit{Name: "Meditar", Frequency: habit.Daily, TargetCount: 1},
			logs:  []habit.Log{logOn(day(10, 10), 1), logOn(day(10, 13), 1), logOn(day(10, 14), 1), logOn(day(10, 15), 2)},
			start: day(10, 10),
			// Días cerrados: 10..15 (6), cumplidos 10, 13, 14, 15; hoy aún no.
			want: HabitSummary{Name: "Meditar", Frequency: habit.Daily, TargetCount: 1, PeriodsMet: 4, PeriodsTotal: 6, CurrentStreak: 3},
		},
		{
			name:  "diario: hoy cumplido cuenta y suma a la racha",
			habit: habit.Habit{Name: "Leer", Frequency: habit.Daily, TargetCount: 2},
			logs:  []habit.Log{logOn(day(10, 14), 1), logOn(day(10, 15), 2), logOn(day(10, 16), 3)},
			start: day(10, 14),
			want:  HabitSummary{Name: "Leer", Frequency: habit.Daily, TargetCount: 2, PeriodsMet: 2, PeriodsTotal: 3, CurrentStreak: 2, CurrentCount: 3},
		},
		{
			name:  "semanal: suma los días de cada semana ISO",
			habit: habit.Habit{Name: "Correr", Frequency: habit.Weekly, TargetCount: 3},
			// Semanas desde el lunes 28/9: [28/9] 3 ✓, [5/10] 1 ✗, [12/10 en curso] 2.
			logs:  []habit.Log{logOn(day(9, 28), 1), logOn(day(9, 30), 2), logOn(day(10, 7), 1), logOn(day(10, 12), 1), logOn(day(10, 15), 1)},
			start: day(10, 1),
			want:  HabitSummary{Name: "Correr", Frequency: habit.Weekly, TargetCount: 3, PeriodsMet: 1, PeriodsTotal: 2, CurrentStreak: 0, CurrentCount: 2},
		},
		{
			name:  "mensual: el mes en curso solo cuenta si ya se cumplió",
			habit: habit.Habit{Name: "Revisar presupuesto", Frequency: habit.Monthly, TargetCount: 1},
			logs:  []habit.Log{logOn(day(9, 30), 1), logOn(day(10, 2), 1)},
			start: day(9, 17),
			want:  HabitSummary{Name: "Revisar presupuesto", Frequency: habit.Monthly, TargetCount: 1, PeriodsMet: 2, PeriodsTotal: 2, CurrentStreak: 2, CurrentCount: 1},
		},
		{
			name:  "creado hoy sin registros",
			habit: habit.Habit{Name: "Agua", Frequency: habit.Daily, TargetCount: 8},
			start: today,
			want:  HabitSummary{Name: "Agua", Frequency: habit.Daily, TargetCount: 8},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SummarizeHabit(tt.habit, tt.logs, tt.start, today)
			if got != tt.want {
				t.Errorf("SummarizeHabit() = %+v, se esperaba %+v", got, tt.want)
			}
		})
	}
}

func TestTruncateName(t *testing.T) {
	long := strings.Repeat("ñ", maxNameLen+10)
	got := truncateName(long)
	if []rune(got)[maxNameLen-1] != '…' || len([]rune(got)) != maxNameLen {
		t.Errorf("truncateName() = %q (%d runas), se esperaban %d terminadas en …", got, len([]rune(got)), maxNameLen)
	}
	if got := truncateName("  Gym \t diario "); got != "Gym diario" {
		t.Errorf("truncateName() = %q", got)
	}
}

func TestSnapshotIsEmpty(t *testing.T) {
	if !(Snapshot{}).IsEmpty() {
		t.Error("un Snapshot sin finanzas ni hábitos debe estar vacío")
	}
	if (Snapshot{Habits: []HabitSummary{{Name: "Leer"}}}).IsEmpty() {
		t.Error("un Snapshot con hábitos no está vacío")
	}
}
