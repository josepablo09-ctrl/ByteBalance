// Package insight contiene las reglas del análisis con IA: qué datos del
// usuario se resumen para el modelo (minimizados y sin identificadores) y cómo
// se valida y sanea el diagnóstico que devuelve.
package insight

import (
	"cmp"
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/finance"
	"bytebalance/internal/domain/habit"
)

// WindowDays es el número de días, hasta hoy inclusive, que cubre el análisis.
// Las finanzas se comparan además con los WindowDays anteriores.
const WindowDays = 30

// Límites del resumen: acotan el tamaño del prompt y lo que sale del sistema.
const (
	// MaxHabits es el número máximo de hábitos que se analizan.
	MaxHabits = 20
	// maxTopExpenses es el número de categorías de gasto que se detallan por moneda.
	maxTopExpenses = 5
	// maxNameLen acota los nombres de categorías y hábitos, que escribe el usuario.
	maxNameLen = 40
)

// UncategorizedName es el nombre con el que se agrupan los gastos sin categoría.
const UncategorizedName = "Sin categoría"

// ErrNotEnoughData indica que el usuario aún no tiene datos que analizar.
var ErrNotEnoughData = errors.New("no hay suficientes datos para el análisis")

// Snapshot es el resumen agregado que se envía al modelo. No contiene
// identificadores, emails, nombres de personas ni descripciones de
// transacciones: solo totales, categorías y la constancia de los hábitos.
type Snapshot struct {
	// From y To son las fechas de calendario (inclusive) del periodo analizado.
	From     time.Time
	To       time.Time
	Finances []CurrencySummary
	Habits   []HabitSummary
}

// IsEmpty indica si no hay nada que analizar.
func (s Snapshot) IsEmpty() bool {
	return len(s.Finances) == 0 && len(s.Habits) == 0
}

// CurrencySummary resume los movimientos de una moneda. Las monedas no se
// mezclan porque no hay conversión entre ellas.
type CurrencySummary struct {
	Currency     domain.Currency
	IncomeCents  int64
	ExpenseCents int64
	// PreviousIncomeCents y PreviousExpenseCents son los del periodo anterior,
	// de la misma duración, para comparar la tendencia.
	PreviousIncomeCents  int64
	PreviousExpenseCents int64
	// Transactions es el número de movimientos del periodo actual.
	Transactions int
	// TopExpenses son las categorías con más gasto en el periodo actual, de mayor a menor.
	TopExpenses []CategoryAmount
}

// CategoryAmount es el total gastado en una categoría.
type CategoryAmount struct {
	Name  string
	Cents int64
}

// SummarizeFinances agrupa por moneda las transacciones del periodo actual y
// del anterior. categoryNames traduce los IDs de categoría a su nombre; las
// transacciones sin categoría (o con una ya eliminada) se agrupan como
// UncategorizedName. Una moneda que solo aparece en el periodo anterior se
// incluye igualmente, para reflejar que dejó de usarse.
func SummarizeFinances(current, previous []finance.Transaction, categoryNames map[domain.ID]string) []CurrencySummary {
	byCurrency := map[domain.Currency]*CurrencySummary{}
	expensesByCategory := map[domain.Currency]map[string]int64{}
	get := func(c domain.Currency) *CurrencySummary {
		s, ok := byCurrency[c]
		if !ok {
			s = &CurrencySummary{Currency: c}
			byCurrency[c] = s
			expensesByCategory[c] = map[string]int64{}
		}
		return s
	}

	for _, t := range current {
		s := get(t.Amount.Currency)
		s.Transactions++
		switch t.Type {
		case finance.Income:
			s.IncomeCents += t.Amount.Cents
		case finance.Expense:
			s.ExpenseCents += t.Amount.Cents
			expensesByCategory[t.Amount.Currency][categoryName(t.CategoryID, categoryNames)] += t.Amount.Cents
		}
	}
	for _, t := range previous {
		s := get(t.Amount.Currency)
		switch t.Type {
		case finance.Income:
			s.PreviousIncomeCents += t.Amount.Cents
		case finance.Expense:
			s.PreviousExpenseCents += t.Amount.Cents
		}
	}

	summaries := make([]CurrencySummary, 0, len(byCurrency))
	for currency, s := range byCurrency {
		for name, cents := range expensesByCategory[currency] {
			s.TopExpenses = append(s.TopExpenses, CategoryAmount{Name: name, Cents: cents})
		}
		slices.SortFunc(s.TopExpenses, func(a, b CategoryAmount) int {
			return cmp.Or(cmp.Compare(b.Cents, a.Cents), strings.Compare(a.Name, b.Name))
		})
		if len(s.TopExpenses) > maxTopExpenses {
			s.TopExpenses = s.TopExpenses[:maxTopExpenses]
		}
		summaries = append(summaries, *s)
	}
	// Primero la moneda con más movimientos: es la principal del usuario.
	slices.SortFunc(summaries, func(a, b CurrencySummary) int {
		return cmp.Or(cmp.Compare(b.Transactions, a.Transactions), strings.Compare(string(a.Currency), string(b.Currency)))
	})
	return summaries
}

func categoryName(id *domain.ID, names map[domain.ID]string) string {
	if id != nil {
		if name, ok := names[*id]; ok {
			return truncateName(name)
		}
	}
	return UncategorizedName
}

// HabitSummary resume la constancia de un hábito activo en el periodo.
type HabitSummary struct {
	Name        string
	Frequency   habit.Frequency
	TargetCount int
	// PeriodsMet y PeriodsTotal cuentan los días, semanas o meses (según la
	// frecuencia) en que se cumplió la meta. El periodo en curso solo cuenta
	// si ya se cumplió: aún puede completarse.
	PeriodsMet   int
	PeriodsTotal int
	// CurrentStreak es el número de periodos consecutivos cumplidos hasta hoy.
	CurrentStreak int
	// CurrentCount es el progreso del periodo en curso.
	CurrentCount int
}

// SummarizeHabit calcula la constancia de h entre start y today (fechas de
// calendario normalizadas con habit.DateOf, start <= today). start suele ser
// el inicio del análisis o, si es posterior, el día en que se creó el hábito.
// logs debe incluir los registros desde habit.PeriodStart(h.Frequency, start).
func SummarizeHabit(h habit.Habit, logs []habit.Log, start, today time.Time) HabitSummary {
	s := HabitSummary{Name: truncateName(h.Name), Frequency: h.Frequency, TargetCount: h.TargetCount}
	target := max(h.TargetCount, 1)

	countIn := func(from, to time.Time) int {
		total := 0
		for _, l := range logs {
			if !l.LoggedOn.Before(from) && l.LoggedOn.Before(to) {
				total += l.Count
			}
		}
		return total
	}

	// met registra, del más antiguo al más reciente, si se cumplió cada periodo cerrado.
	var met []bool
	current := habit.PeriodStart(h.Frequency, today)
	for p := habit.PeriodStart(h.Frequency, start); p.Before(current); p = nextPeriod(h.Frequency, p) {
		met = append(met, countIn(p, nextPeriod(h.Frequency, p)) >= target)
	}
	s.CurrentCount = countIn(current, nextPeriod(h.Frequency, current))
	if s.CurrentCount >= target {
		met = append(met, true)
	}

	s.PeriodsTotal = len(met)
	for _, ok := range met {
		if ok {
			s.PeriodsMet++
		}
	}
	for i := len(met) - 1; i >= 0 && met[i]; i-- {
		s.CurrentStreak++
	}
	return s
}

// nextPeriod devuelve el inicio del periodo siguiente al que empieza en p.
func nextPeriod(f habit.Frequency, p time.Time) time.Time {
	switch f {
	case habit.Weekly:
		return p.AddDate(0, 0, 7)
	case habit.Monthly:
		return p.AddDate(0, 1, 0)
	default:
		return p.AddDate(0, 0, 1)
	}
}

// truncateName limpia y acota un nombre escrito por el usuario.
func truncateName(name string) string {
	return truncate(strings.Join(strings.Fields(name), " "), maxNameLen)
}

// truncate corta s a max runas, terminando en "…" si se recortó.
func truncate(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return strings.TrimSpace(string(runes[:max-1])) + "…"
}
