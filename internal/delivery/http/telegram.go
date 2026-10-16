package http

import (
	"context"
	"net/http"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/usecase"
)

// Límite de enlaces de vinculación por usuario: 5 seguidos y luego 1 cada 30 s.
// Cada enlace nuevo invalida el anterior, así que más no tienen utilidad.
const (
	telegramLinkBurst    = 5
	telegramLinkInterval = 30 * time.Second
)

// TelegramService son los casos de uso de Telegram que expone la API.
type TelegramService interface {
	BotUsername() string
	CreateLink(ctx context.Context, userID domain.ID) (usecase.TelegramLink, error)
	Unlink(ctx context.Context, userID domain.ID) error
	SendTest(ctx context.Context, userID domain.ID) error
}

type telegramStatusResponse struct {
	// Enabled es false si el servidor no tiene bot configurado.
	Enabled     bool   `json:"enabled"`
	Linked      bool   `json:"linked"`
	BotUsername string `json:"bot_username,omitempty"`
}

type telegramLinkResponse struct {
	Code      string    `json:"code"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

// telegramStatus atiende GET /api/v1/telegram.
func (a *api) telegramStatus(w http.ResponseWriter, r *http.Request) {
	u, err := a.Users.Profile(r.Context(), userIDFrom(r))
	if err != nil {
		respondError(w, r, err)
		return
	}
	resp := telegramStatusResponse{Enabled: a.Telegram != nil, Linked: u.HasTelegram()}
	if a.Telegram != nil {
		resp.BotUsername = a.Telegram.BotUsername()
	}
	writeJSON(w, http.StatusOK, resp)
}

// createTelegramLink atiende POST /api/v1/telegram/link: genera el enlace de
// un solo uso para abrir el bot y vincular la cuenta.
func (a *api) createTelegramLink(w http.ResponseWriter, r *http.Request) {
	if !a.requireTelegram(w) {
		return
	}
	userID := userIDFrom(r)
	if ok, wait := a.telegramLinkLimits.allow(userID.String()); !ok {
		tooManyRequests(w, r, wait, "demasiados enlaces generados; intenta de nuevo en unos segundos")
		return
	}
	link, err := a.Telegram.CreateLink(r.Context(), userID)
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, telegramLinkResponse{Code: link.Code, URL: link.URL, ExpiresAt: link.ExpiresAt})
}

// deleteTelegramLink atiende DELETE /api/v1/telegram/link: desvincula el chat.
func (a *api) deleteTelegramLink(w http.ResponseWriter, r *http.Request) {
	if !a.requireTelegram(w) {
		return
	}
	if err := a.Telegram.Unlink(r.Context(), userIDFrom(r)); err != nil {
		respondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// sendTelegramTest atiende POST /api/v1/telegram/test.
func (a *api) sendTelegramTest(w http.ResponseWriter, r *http.Request) {
	if !a.requireTelegram(w) {
		return
	}
	if err := a.Telegram.SendTest(r.Context(), userIDFrom(r)); err != nil {
		respondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// requireTelegram responde 503 si el servidor no tiene bot configurado.
func (a *api) requireTelegram(w http.ResponseWriter) bool {
	if a.Telegram == nil {
		writeError(w, http.StatusServiceUnavailable, "telegram_disabled", "la integración con Telegram no está configurada en el servidor")
		return false
	}
	return true
}
