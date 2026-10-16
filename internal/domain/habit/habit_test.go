package habit

import (
	"errors"
	"strings"
	"testing"
	"time"

	"bytebalance/internal/domain"
)

var (
	userA  = domain.MustParseID("11111111-1111-1111-1111-111111111111")
	habitA = domain.MustParseID("33333333-3333-3333-3333-333333333333")
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		params  Params
		wantErr bool
		check   func(t *testing.T, h *Habit)
	}{
		{
			name:   "aplica valores por defecto",
			params: Params{UserID: userA, Name: " Leer "},
			check: func(t *testing.T, h *Habit) {
				if h.Name != "Leer" || h.Frequency != Daily || h.TargetCount != 1 || !h.IsActive {
					t.Errorf("valores por defecto incorrectos: %+v", h)
				}
			},
		},
		{
			name:   "con recordatorio",
			params: Params{UserID: userA, Name: "Correr", Frequency: Weekly, TargetCount: 3, ReminderTime: &TimeOfDay{Hour: 6, Minute: 30}},
		},
		{name: "sin usuario", params: Params{Name: "Leer"}, wantErr: true},
		{name: "nombre vacío", params: Params{UserID: userA, Name: " "}, wantErr: true},
		{name: "nombre demasiado largo", params: Params{UserID: userA, Name: strings.Repeat("x", 101)}, wantErr: true},
		{name: "frecuencia inválida", params: Params{UserID: userA, Name: "Leer", Frequency: "yearly"}, wantErr: true},
		{name: "meta negativa", params: Params{UserID: userA, Name: "Leer", TargetCount: -1}, wantErr: true},
		{name: "meta máxima", params: Params{UserID: userA, Name: "Leer", TargetCount: MaxTargetCount}},
		{name: "meta por encima del máximo", params: Params{UserID: userA, Name: "Leer", TargetCount: MaxTargetCount + 1}, wantErr: true},
		// Se cuentan caracteres, no bytes: 500 "ñ" ocupan 1000 bytes y son válidas.
		{name: "descripción en el límite", params: Params{UserID: userA, Name: "Leer", Description: strings.Repeat("ñ", 500)}},
		{name: "descripción demasiado larga", params: Params{UserID: userA, Name: "Leer", Description: strings.Repeat("x", 501)}, wantErr: true},
		{name: "hora inválida", params: Params{UserID: userA, Name: "Leer", ReminderTime: &TimeOfDay{Hour: 24}}, wantErr: true},
		{name: "minuto inválido", params: Params{UserID: userA, Name: "Leer", ReminderTime: &TimeOfDay{Hour: 8, Minute: 60}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, err := New(tt.params)
			if tt.wantErr {
				if !errors.Is(err, domain.ErrInvalidInput) {
					t.Fatalf("New() error = %v, se esperaba ErrInvalidInput", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("New() error inesperado: %v", err)
			}
			if tt.check != nil {
				tt.check(t, h)
			}
		})
	}
}

func TestRecord(t *testing.T) {
	costaRica := time.FixedZone("CST", -6*60*60)
	// 23:30 del 10 de octubre en Costa Rica ya es 11 de octubre en UTC.
	lateNight := time.Date(2026, 10, 10, 23, 30, 0, 0, costaRica)

	tests := []struct {
		name     string
		habit    Habit
		count    int
		wantErr  error
		wantDate time.Time
	}{
		{
			name:     "usa la fecha local del usuario",
			habit:    Habit{ID: habitA, UserID: userA, IsActive: true},
			count:    2,
			wantDate: time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "cantidad cero se toma como 1",
			habit:    Habit{ID: habitA, UserID: userA, IsActive: true},
			wantDate: time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC),
		},
		{name: "hábito inactivo", habit: Habit{ID: habitA, UserID: userA}, wantErr: ErrInactive},
		{name: "hábito sin persistir", habit: Habit{UserID: userA, IsActive: true}, wantErr: domain.ErrInvalidInput},
		{name: "cantidad negativa", habit: Habit{ID: habitA, UserID: userA, IsActive: true}, count: -1, wantErr: domain.ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, err := tt.habit.Record(lateNight, tt.count, " nota ")
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Record() error = %v, se esperaba %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Record() error inesperado: %v", err)
			}
			if !l.LoggedOn.Equal(tt.wantDate) {
				t.Errorf("LoggedOn = %v, se esperaba %v", l.LoggedOn, tt.wantDate)
			}
			if l.Count < 1 || l.Note != "nota" {
				t.Errorf("Log inesperado: %+v", l)
			}
		})
	}
}

func TestErrInactiveIsInvalidInput(t *testing.T) {
	if !errors.Is(ErrInactive, domain.ErrInvalidInput) {
		t.Fatal("ErrInactive debe envolver domain.ErrInvalidInput")
	}
}

func TestNewLogLimits(t *testing.T) {
	day := time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		count   int
		note    string
		wantErr bool
	}{
		{name: "cantidad máxima", count: MaxLogCount},
		{name: "cantidad por encima del máximo", count: MaxLogCount + 1, wantErr: true},
		{name: "nota en el límite", count: 1, note: strings.Repeat("ñ", 280)},
		{name: "nota demasiado larga", count: 1, note: strings.Repeat("x", 281), wantErr: true},
		{name: "los espacios de la nota no cuentan", count: 1, note: "  " + strings.Repeat("x", 280) + "  "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewLog(habitA, day, tt.count, tt.note)
			if tt.wantErr {
				if !errors.Is(err, domain.ErrInvalidInput) {
					t.Fatalf("NewLog() error = %v, se esperaba ErrInvalidInput", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewLog() error inesperado: %v", err)
			}
		})
	}
}

func TestPeriodStart(t *testing.T) {
	date := func(m time.Month, d int) time.Time { return time.Date(2026, m, d, 0, 0, 0, 0, time.UTC) }
	tests := []struct {
		name string
		f    Frequency
		day  time.Time
		want time.Time
	}{
		{name: "diario", f: Daily, day: date(10, 16), want: date(10, 16)},
		{name: "semanal un viernes", f: Weekly, day: date(10, 16), want: date(10, 12)},
		{name: "semanal un lunes", f: Weekly, day: date(10, 12), want: date(10, 12)},
		{name: "semanal un domingo", f: Weekly, day: date(10, 18), want: date(10, 12)},
		{name: "semana que cruza el mes", f: Weekly, day: date(11, 1), want: date(10, 26)},
		{name: "mensual", f: Monthly, day: date(10, 16), want: date(10, 1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PeriodStart(tt.f, tt.day); !got.Equal(tt.want) {
				t.Errorf("PeriodStart() = %s, se esperaba %s", got.Format(time.DateOnly), tt.want.Format(time.DateOnly))
			}
		})
	}
}
