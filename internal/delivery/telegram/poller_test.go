package telegram

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	tgapi "bytebalance/internal/integration/telegram"
)

// scriptedSource devuelve una respuesta por llamada y cancela el contexto
// cuando se acaban, para terminar Run.
type scriptedSource struct {
	mu      sync.Mutex
	steps   []step
	offsets []int64
	cancel  context.CancelFunc
}

type step struct {
	updates []tgapi.Update
	err     error
}

func (s *scriptedSource) GetUpdates(ctx context.Context, offset int64, _ time.Duration) ([]tgapi.Update, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.offsets = append(s.offsets, offset)
	if len(s.steps) == 0 {
		s.cancel()
		<-ctx.Done()
		return nil, ctx.Err()
	}
	next := s.steps[0]
	s.steps = s.steps[1:]
	return next.updates, next.err
}

type recordingHandler struct {
	texts []string
	err   error
}

func (h *recordingHandler) HandleMessage(_ context.Context, _ int64, _ string, text string) error {
	h.texts = append(h.texts, text)
	return h.err
}

func message(id int64, text string) tgapi.Update {
	return tgapi.Update{UpdateID: id, Message: &tgapi.Message{Chat: tgapi.Chat{ID: 1, Type: "private"}, Text: text}}
}

func runPoller(t *testing.T, steps []step, handler *recordingHandler) *scriptedSource {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	source := &scriptedSource{steps: steps, cancel: cancel}
	p := NewPoller(source, handler)
	p.minBackoff, p.maxBackoff = time.Millisecond, 4*time.Millisecond

	done := make(chan struct{})
	go func() {
		p.Run(ctx)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run no terminó")
	}
	return source
}

func TestPollerDeliversMessagesAndAdvancesOffset(t *testing.T) {
	handler := &recordingHandler{}
	source := runPoller(t, []step{
		{updates: []tgapi.Update{message(10, "/start abc"), {UpdateID: 11}, message(12, "hola")}},
		{updates: []tgapi.Update{message(13, "/stop")}},
	}, handler)

	if want := []string{"/start abc", "hola", "/stop"}; !slices.Equal(handler.texts, want) {
		t.Errorf("mensajes = %v, se esperaba %v", handler.texts, want)
	}
	if want := []int64{0, 13, 14}; !slices.Equal(source.offsets, want) {
		t.Errorf("offsets = %v, se esperaba %v", source.offsets, want)
	}
}

func TestPollerRetriesAfterErrorsAndHandlerFailures(t *testing.T) {
	handler := &recordingHandler{err: errors.New("falló")}
	source := runPoller(t, []step{
		{err: errors.New("red caída")},
		{err: &tgapi.APIError{Method: "getUpdates", Code: 429, RetryAfter: time.Millisecond}},
		{updates: []tgapi.Update{message(5, "uno"), message(6, "dos")}},
	}, handler)

	// Un handler que falla no bloquea los mensajes siguientes ni repite el mismo.
	if want := []string{"uno", "dos"}; !slices.Equal(handler.texts, want) {
		t.Errorf("mensajes = %v, se esperaba %v", handler.texts, want)
	}
	if want := []int64{0, 0, 0, 7}; !slices.Equal(source.offsets, want) {
		t.Errorf("offsets = %v, se esperaba %v", source.offsets, want)
	}
}
