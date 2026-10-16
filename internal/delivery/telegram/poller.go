// Package telegram recibe los mensajes del bot de Telegram por long polling y
// los entrega a los casos de uso. Es el equivalente de delivery/http para el bot.
package telegram

import (
	"context"
	"errors"
	"log/slog"
	"time"

	tgapi "bytebalance/internal/integration/telegram"
)

// Valores por defecto del poller.
const (
	defaultWait       = 30 * time.Second
	defaultMinBackoff = time.Second
	defaultMaxBackoff = time.Minute
)

// updatesSource obtiene novedades del bot (long polling).
type updatesSource interface {
	GetUpdates(ctx context.Context, offset int64, wait time.Duration) ([]tgapi.Update, error)
}

// MessageHandler atiende un mensaje de texto recibido por el bot.
type MessageHandler interface {
	HandleMessage(ctx context.Context, chatID int64, chatType, text string) error
}

// Poller pide novedades a Telegram en bucle y entrega los mensajes al handler.
//
// Solo debe correr una instancia por bot: Telegram responde 409 si dos procesos
// piden novedades a la vez. Con varias réplicas hay que pasar a webhooks (ver
// docs/adr/0005-integracion-con-telegram.md).
type Poller struct {
	source     updatesSource
	handler    MessageHandler
	wait       time.Duration
	minBackoff time.Duration
	maxBackoff time.Duration
}

// NewPoller crea un Poller con esperas de 30 s y reintentos de 1 s a 1 min.
func NewPoller(source updatesSource, handler MessageHandler) *Poller {
	return &Poller{
		source:     source,
		handler:    handler,
		wait:       defaultWait,
		minBackoff: defaultMinBackoff,
		maxBackoff: defaultMaxBackoff,
	}
}

// Run recibe y atiende mensajes hasta que ctx termine. Ante un error espera con
// backoff exponencial (o lo que indique Telegram) y vuelve a intentar.
func (p *Poller) Run(ctx context.Context) {
	var offset int64
	backoff := p.minBackoff
	for {
		updates, err := p.source.GetUpdates(ctx, offset, p.wait)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			delay := backoff
			var apiErr *tgapi.APIError
			if errors.As(err, &apiErr) && apiErr.RetryAfter > 0 {
				delay = apiErr.RetryAfter
			}
			slog.WarnContext(ctx, "recibir mensajes de Telegram", "error", err, "reintento_en", delay.String())
			if !sleep(ctx, delay) {
				return
			}
			backoff = min(backoff*2, p.maxBackoff)
			continue
		}
		backoff = p.minBackoff

		for _, u := range updates {
			// Se avanza el offset aunque falle el handler: reintentar un mensaje
			// que falla siempre bloquearía a todos los siguientes.
			offset = u.UpdateID + 1
			if u.Message == nil || u.Message.Text == "" {
				continue
			}
			if err := p.handler.HandleMessage(ctx, u.Message.Chat.ID, u.Message.Chat.Type, u.Message.Text); err != nil {
				// No se registra el chat ni el texto: pueden identificar al usuario.
				slog.ErrorContext(ctx, "atender mensaje de Telegram", "error", err)
			}
		}
	}
}

// sleep espera d o hasta que ctx termine; devuelve false si ctx terminó.
func sleep(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
