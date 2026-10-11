// Package habit modela los hábitos de un usuario y su registro de cumplimiento.
package habit

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"bytebalance/internal/domain"
)

const maxNameLen = 100

// ErrInactive indica que se intentó registrar progreso en un hábito pausado.
var ErrInactive = fmt.Errorf("%w: el hábito está inactivo", domain.ErrInvalidInput)

// Frequency es el periodo en el que se evalúa la meta de un hábito.
type Frequency string

// Frecuencias admitidas.
const (
	Daily   Frequency = "daily"
	Weekly  Frequency = "weekly"
	Monthly Frequency = "monthly"
)

// Valid indica si f es una frecuencia conocida.
func (f Frequency) Valid() bool {
	return f == Daily || f == Weekly || f == Monthly
}

// TimeOfDay es una hora del día sin fecha ni zona horaria (columna TIME).
// Se interpreta en la zona horaria del usuario.
type TimeOfDay struct {
	Hour   int
	Minute int
}

// Valid indica si t es una hora entre 00:00 y 23:59.
func (t TimeOfDay) Valid() bool {
	return t.Hour >= 0 && t.Hour < 24 && t.Minute >= 0 && t.Minute < 60
}

// String devuelve la hora en formato HH:MM.
func (t TimeOfDay) String() string {
	return fmt.Sprintf("%02d:%02d", t.Hour, t.Minute)
}

// Habit es una actividad que el usuario quiere repetir TargetCount veces por periodo.
type Habit struct {
	ID          domain.ID
	UserID      domain.ID
	Name        string
	Description string
	Frequency   Frequency
	TargetCount int
	// ReminderTime es nil si el usuario no quiere recordatorios.
	ReminderTime *TimeOfDay
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

// Params agrupa los datos necesarios para crear un Habit.
type Params struct {
	UserID      domain.ID
	Name        string
	Description string
	// Frequency vacía equivale a Daily.
	Frequency Frequency
	// TargetCount cero equivale a 1.
	TargetCount  int
	ReminderTime *TimeOfDay
}

// New devuelve un Habit activo y válido sin persistir.
func New(p Params) (*Habit, error) {
	h := &Habit{
		UserID:       p.UserID,
		Name:         strings.TrimSpace(p.Name),
		Description:  strings.TrimSpace(p.Description),
		Frequency:    p.Frequency,
		TargetCount:  p.TargetCount,
		ReminderTime: p.ReminderTime,
		IsActive:     true,
	}
	if h.Frequency == "" {
		h.Frequency = Daily
	}
	if h.TargetCount == 0 {
		h.TargetCount = 1
	}
	if err := h.Validate(); err != nil {
		return nil, err
	}
	return h, nil
}

// Validate comprueba las invariantes del hábito.
func (h *Habit) Validate() error {
	if h.UserID.IsZero() {
		return fmt.Errorf("%w: el hábito debe pertenecer a un usuario", domain.ErrInvalidInput)
	}
	if h.Name == "" || utf8.RuneCountInString(h.Name) > maxNameLen {
		return fmt.Errorf("%w: el nombre del hábito es obligatorio y admite hasta %d caracteres", domain.ErrInvalidInput, maxNameLen)
	}
	if !h.Frequency.Valid() {
		return fmt.Errorf("%w: frecuencia %q inválida", domain.ErrInvalidInput, h.Frequency)
	}
	if h.TargetCount <= 0 {
		return fmt.Errorf("%w: la meta debe ser mayor que cero", domain.ErrInvalidInput)
	}
	if h.ReminderTime != nil && !h.ReminderTime.Valid() {
		return fmt.Errorf("%w: hora de recordatorio inválida", domain.ErrInvalidInput)
	}
	return nil
}

// Activate reanuda el hábito.
func (h *Habit) Activate() {
	h.IsActive = true
}

// Deactivate pausa el hábito: deja de admitir registros y recordatorios.
func (h *Habit) Deactivate() {
	h.IsActive = false
}

// Record crea el registro de cumplimiento del hábito para el día de loggedOn.
// loggedOn debe expresarse en la zona horaria del usuario.
func (h *Habit) Record(loggedOn time.Time, count int, note string) (*Log, error) {
	if !h.IsActive {
		return nil, ErrInactive
	}
	if h.ID.IsZero() {
		return nil, fmt.Errorf("%w: el hábito no está persistido", domain.ErrInvalidInput)
	}
	return NewLog(h.ID, loggedOn, count, note)
}
