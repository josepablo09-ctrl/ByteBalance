// Package scheduler ejecuta los trabajos periódicos del backend.
package scheduler

import (
	"context"
	"log/slog"
	"time"
)

// defaultTick es cada cuánto se revisa si empezó un minuto nuevo. Es menor que
// un minuto para no saltarse ninguno aunque el proceso se retrase un poco.
const defaultTick = 15 * time.Second

// reminderSender envía los recordatorios que tocan en un minuto.
type reminderSender interface {
	SendDue(ctx context.Context, at time.Time) (int, error)
}

// Reminders envía los recordatorios de hábitos una vez por minuto.
//
// Solo debe correr en una instancia: con varias réplicas cada una enviaría los
// mismos recordatorios (ver docs/adr/0005-integracion-con-telegram.md). Los
// minutos en que el proceso estuvo detenido no se recuperan.
type Reminders struct {
	sender reminderSender
	now    func() time.Time
	tick   time.Duration
}

// NewReminders crea el trabajo de recordatorios. Si now es nil se usa time.Now.
func NewReminders(sender reminderSender, now func() time.Time) *Reminders {
	if now == nil {
		now = time.Now
	}
	return &Reminders{sender: sender, now: now, tick: defaultTick}
}

// Run procesa cada minuto exactamente una vez hasta que ctx termine.
func (r *Reminders) Run(ctx context.Context) {
	ticker := time.NewTicker(r.tick)
	defer ticker.Stop()
	var last time.Time
	for {
		if minute := r.now().UTC().Truncate(time.Minute); !minute.Equal(last) {
			last = minute
			r.runOnce(ctx, minute)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (r *Reminders) runOnce(ctx context.Context, minute time.Time) {
	sent, err := r.sender.SendDue(ctx, minute)
	if err != nil {
		slog.ErrorContext(ctx, "enviar recordatorios de hábitos", "error", err, "enviados", sent)
		return
	}
	if sent > 0 {
		slog.InfoContext(ctx, "recordatorios de hábitos enviados", "enviados", sent)
	}
}
