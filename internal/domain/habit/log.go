package habit

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"bytebalance/internal/domain"
)

// Log es el cumplimiento de un hábito en un día concreto. Hay como máximo un
// Log por hábito y día.
type Log struct {
	ID      domain.ID
	HabitID domain.ID
	// LoggedOn es una fecha de calendario normalizada con DateOf.
	LoggedOn  time.Time
	Count     int
	Note      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewLog devuelve un Log válido sin persistir. Si count es cero se toma como 1.
func NewLog(habitID domain.ID, loggedOn time.Time, count int, note string) (*Log, error) {
	if count == 0 {
		count = 1
	}
	l := &Log{
		HabitID:  habitID,
		LoggedOn: DateOf(loggedOn),
		Count:    count,
		Note:     strings.TrimSpace(note),
	}
	if err := l.Validate(); err != nil {
		return nil, err
	}
	return l, nil
}

// Validate comprueba las invariantes del registro.
func (l *Log) Validate() error {
	if l.HabitID.IsZero() {
		return fmt.Errorf("%w: el registro debe pertenecer a un hábito", domain.ErrInvalidInput)
	}
	if l.LoggedOn.IsZero() {
		return fmt.Errorf("%w: falta la fecha del registro", domain.ErrInvalidInput)
	}
	if l.Count <= 0 || l.Count > MaxLogCount {
		return fmt.Errorf("%w: la cantidad debe estar entre 1 y %d", domain.ErrInvalidInput, MaxLogCount)
	}
	if utf8.RuneCountInString(l.Note) > maxNoteLen {
		return fmt.Errorf("%w: la nota admite hasta %d caracteres", domain.ErrInvalidInput, maxNoteLen)
	}
	return nil
}

// DateOf conserva solo la fecha de calendario de t (en su propia zona horaria)
// y la devuelve a medianoche UTC, que es como se representa una columna DATE.
func DateOf(t time.Time) time.Time {
	if t.IsZero() {
		return time.Time{}
	}
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// PeriodStart devuelve el primer día del periodo de la meta que contiene day
// (una fecha normalizada con DateOf): el mismo día, el lunes de su semana ISO o
// el primero de su mes.
func PeriodStart(f Frequency, day time.Time) time.Time {
	switch f {
	case Weekly:
		return day.AddDate(0, 0, -((int(day.Weekday()) + 6) % 7))
	case Monthly:
		return time.Date(day.Year(), day.Month(), 1, 0, 0, 0, 0, time.UTC)
	default:
		return day
	}
}
