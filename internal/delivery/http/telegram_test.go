package http

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/user"
	"bytebalance/internal/usecase"
)

// stubTelegram simula el servicio de Telegram; los errores se configuran por test.
type stubTelegram struct {
	unlinkErr error
	testErr   error
	calls     *[]string
}

func (s stubTelegram) record(call string) {
	if s.calls != nil {
		*s.calls = append(*s.calls, call)
	}
}

func (stubTelegram) BotUsername() string { return "ByteBalanceBot" }

func (s stubTelegram) CreateLink(_ context.Context, userID domain.ID) (usecase.TelegramLink, error) {
	s.record("link")
	if userID != testUserID {
		return usecase.TelegramLink{}, fmt.Errorf("usuario inesperado %v", userID)
	}
	return usecase.TelegramLink{
		Code:      "codigo123",
		URL:       "https://t.me/ByteBalanceBot?start=codigo123",
		ExpiresAt: time.Date(2026, 10, 16, 12, 10, 0, 0, time.UTC),
	}, nil
}

func (s stubTelegram) Unlink(context.Context, domain.ID) error {
	s.record("unlink")
	return s.unlinkErr
}

func (s stubTelegram) SendTest(context.Context, domain.ID) error {
	s.record("test")
	return s.testErr
}

func linkedUser() *user.User {
	u := sampleUser()
	chat := int64(777)
	u.TelegramChatID = &chat
	return u
}

func TestTelegramStatus(t *testing.T) {
	profile := func(u *user.User) stubUsers {
		return stubUsers{profile: func(domain.ID) (*user.User, error) { return u, nil }}
	}
	tests := []struct {
		name     string
		services Services
		want     map[string]any
	}{
		{
			name:     "sin bot configurado",
			services: Services{Users: profile(sampleUser())},
			want:     map[string]any{"enabled": false, "linked": false, "bot_username": nil},
		},
		{
			name:     "con bot y sin vincular",
			services: Services{Users: profile(sampleUser()), Telegram: stubTelegram{}},
			want:     map[string]any{"enabled": true, "linked": false, "bot_username": "ByteBalanceBot"},
		},
		{
			name:     "vinculado",
			services: Services{Users: profile(linkedUser()), Telegram: stubTelegram{}},
			want:     map[string]any{"enabled": true, "linked": true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(newTestRouter(tt.services), http.MethodGet, "/api/v1/telegram", "", true)
			expectStatus(t, rec, http.StatusOK)
			body := decodeJSONBody(t, rec)
			for k, v := range tt.want {
				if body[k] != v {
					t.Errorf("%s = %v, se esperaba %v", k, body[k], v)
				}
			}
			// El chat ID nunca se expone al cliente.
			if _, ok := body["chat_id"]; ok {
				t.Error("la respuesta no debe incluir el chat ID")
			}
		})
	}
}

func TestTelegramDisabledEndpoints(t *testing.T) {
	h := newTestRouter(Services{})
	for _, ep := range []struct{ method, path string }{
		{http.MethodPost, "/api/v1/telegram/link"},
		{http.MethodDelete, "/api/v1/telegram/link"},
		{http.MethodPost, "/api/v1/telegram/test"},
	} {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			rec := do(h, ep.method, ep.path, "", true)
			expectStatus(t, rec, http.StatusServiceUnavailable)
			if code := errorCode(t, rec); code != "telegram_disabled" {
				t.Errorf("code = %q", code)
			}
		})
	}
}

func TestCreateTelegramLink(t *testing.T) {
	var calls []string
	h := newTestRouter(Services{Telegram: stubTelegram{calls: &calls}})

	rec := do(h, http.MethodPost, "/api/v1/telegram/link", "", true)
	expectStatus(t, rec, http.StatusCreated)
	body := decodeJSONBody(t, rec)
	if body["url"] != "https://t.me/ByteBalanceBot?start=codigo123" || body["code"] != "codigo123" || body["expires_at"] != "2026-10-16T12:10:00Z" {
		t.Errorf("respuesta = %v", body)
	}

	// Tras la ráfaga permitida, los enlaces nuevos se limitan.
	for range telegramLinkBurst - 1 {
		expectStatus(t, do(h, http.MethodPost, "/api/v1/telegram/link", "", true), http.StatusCreated)
	}
	limited := do(h, http.MethodPost, "/api/v1/telegram/link", "", true)
	expectStatus(t, limited, http.StatusTooManyRequests)
	if limited.Header().Get("Retry-After") == "" {
		t.Error("falta Retry-After")
	}
	if len(calls) != telegramLinkBurst {
		t.Errorf("CreateLink llamado %d veces, se esperaban %d", len(calls), telegramLinkBurst)
	}
}

func TestTelegramUnlinkAndTest(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		svc        stubTelegram
		wantStatus int
		wantCode   string
	}{
		{name: "desvincular", method: http.MethodDelete, path: "/api/v1/telegram/link", wantStatus: http.StatusNoContent},
		{name: "mensaje de prueba", method: http.MethodPost, path: "/api/v1/telegram/test", wantStatus: http.StatusNoContent},
		{
			name: "mensaje de prueba sin vincular", method: http.MethodPost, path: "/api/v1/telegram/test",
			svc: stubTelegram{testErr: usecase.ErrTelegramNotLinked}, wantStatus: http.StatusConflict, wantCode: "telegram_not_linked",
		},
		{
			name: "Telegram caído", method: http.MethodPost, path: "/api/v1/telegram/test",
			svc: stubTelegram{testErr: fmt.Errorf("enviar mensaje de prueba: %w", context.DeadlineExceeded)}, wantStatus: http.StatusInternalServerError, wantCode: "internal",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(newTestRouter(Services{Telegram: tt.svc}), tt.method, tt.path, "", true)
			expectStatus(t, rec, tt.wantStatus)
			if tt.wantCode != "" {
				if code := errorCode(t, rec); code != tt.wantCode {
					t.Errorf("code = %q, se esperaba %q", code, tt.wantCode)
				}
			}
		})
	}
}

func TestTelegramEndpointsRequireAuth(t *testing.T) {
	h := newTestRouter(Services{Telegram: stubTelegram{}})
	for _, ep := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/telegram"},
		{http.MethodPost, "/api/v1/telegram/link"},
		{http.MethodDelete, "/api/v1/telegram/link"},
		{http.MethodPost, "/api/v1/telegram/test"},
	} {
		expectStatus(t, do(h, ep.method, ep.path, "", false), http.StatusUnauthorized)
	}
}
