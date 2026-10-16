package scheduler

import (
	"context"
	"sync"
	"testing"
	"time"
)

// recordingSender registra los minutos procesados.
type recordingSender struct {
	mu      sync.Mutex
	minutes []time.Time
}

func (s *recordingSender) SendDue(_ context.Context, at time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.minutes = append(s.minutes, at)
	return 0, nil
}

func TestRemindersProcessesEachMinuteOnce(t *testing.T) {
	start := time.Date(2026, 10, 16, 13, 29, 50, 0, time.UTC)
	// Cada tick avanza el reloj 20 s: 13:29:50, 13:30:10, 13:30:30, 13:30:50, 13:31:10…
	var mu sync.Mutex
	calls := 0
	now := func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		t := start.Add(time.Duration(calls) * 20 * time.Second)
		calls++
		return t
	}

	sender := &recordingSender{}
	r := NewReminders(sender, now)
	r.tick = time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		r.Run(ctx)
		close(done)
	}()
	deadline := time.After(5 * time.Second)
	for {
		sender.mu.Lock()
		n := len(sender.minutes)
		sender.mu.Unlock()
		if n >= 3 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("no se procesaron 3 minutos a tiempo")
		case <-time.After(time.Millisecond):
		}
	}
	cancel()
	<-done

	sender.mu.Lock()
	defer sender.mu.Unlock()
	want := []string{"13:29", "13:30", "13:31"}
	for i, w := range want {
		if got := sender.minutes[i].Format("15:04"); got != w || sender.minutes[i].Second() != 0 {
			t.Errorf("minuto %d = %s, se esperaba %s:00", i, sender.minutes[i].Format(time.TimeOnly), w)
		}
	}
}
