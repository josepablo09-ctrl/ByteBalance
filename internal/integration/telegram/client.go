// Package telegram es el cliente de la Telegram Bot API.
//
// El token del bot forma parte de cada URL de la API, por eso ningún error que
// devuelve este paquete incluye la URL: se registran sin exponer el secreto.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// requestTimeout acota cada llamada que no es long polling.
const requestTimeout = 10 * time.Second

// maxResponseBytes limita el tamaño de las respuestas leídas (1 MiB).
const maxResponseBytes = 1 << 20

// Bot identifica al bot autenticado.
type Bot struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

// Update es una novedad recibida por el bot. Solo se modelan los mensajes.
type Update struct {
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message"`
}

// Message es un mensaje de texto recibido en un chat.
type Message struct {
	Chat Chat   `json:"chat"`
	Text string `json:"text"`
}

// Chat es la conversación de la que proviene un mensaje.
type Chat struct {
	ID int64 `json:"id"`
	// Type es "private", "group", "supergroup" o "channel".
	Type string `json:"type"`
}

// APIError es un error devuelto por la Bot API (ok=false).
type APIError struct {
	Method      string
	Code        int
	Description string
	// RetryAfter es la espera indicada por Telegram ante un límite de envío (429).
	RetryAfter time.Duration
}

func (e *APIError) Error() string {
	return fmt.Sprintf("telegram %s: %d %s", e.Method, e.Code, e.Description)
}

// Client llama a la Bot API con el token de un bot.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// NewClient crea un cliente para baseURL (p. ej. https://api.telegram.org).
// Las llamadas de long polling se acotan con el contexto, no con un timeout
// global, por eso el http.Client no lo define.
func NewClient(baseURL, token string) *Client {
	return &Client{baseURL: baseURL, token: token, http: &http.Client{}}
}

// GetMe devuelve el bot autenticado. Sirve para validar el token al arrancar.
func (c *Client) GetMe(ctx context.Context) (Bot, error) {
	var bot Bot
	err := c.call(ctx, "getMe", nil, requestTimeout, &bot)
	return bot, err
}

// SendMessage envía un mensaje de texto plano al chat.
func (c *Client) SendMessage(ctx context.Context, chatID int64, text string) error {
	body := map[string]any{"chat_id": chatID, "text": text, "link_preview_options": map[string]bool{"is_disabled": true}}
	return c.call(ctx, "sendMessage", body, requestTimeout, nil)
}

// GetUpdates espera hasta wait novedades a partir de offset (long polling).
func (c *Client) GetUpdates(ctx context.Context, offset int64, wait time.Duration) ([]Update, error) {
	body := map[string]any{
		"offset":          offset,
		"timeout":         int(wait.Seconds()),
		"allowed_updates": []string{"message"},
	}
	var updates []Update
	// Se deja margen sobre el tiempo de espera del servidor.
	err := c.call(ctx, "getUpdates", body, wait+requestTimeout, &updates)
	return updates, err
}

// apiResponse es el sobre común de las respuestas de la Bot API.
type apiResponse struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result"`
	ErrorCode   int             `json:"error_code"`
	Description string          `json:"description"`
	Parameters  struct {
		RetryAfter int `json:"retry_after"`
	} `json:"parameters"`
}

func (c *Client) call(ctx context.Context, method string, body any, timeout time.Duration, result any) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	payload := []byte("{}")
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			return fmt.Errorf("telegram %s: codificar petición: %w", method, err)
		}
	}
	endpoint := c.baseURL + "/bot" + c.token + "/" + method
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("telegram %s: crear petición: %w", method, redact(err))
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("telegram %s: %w", method, redact(err))
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.WarnContext(ctx, "cerrar respuesta de Telegram", "method", method, "error", err)
		}
	}()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return fmt.Errorf("telegram %s: leer respuesta: %w", method, err)
	}
	var envelope apiResponse
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("telegram %s: respuesta inválida (HTTP %d)", method, resp.StatusCode)
	}
	if !envelope.OK {
		return &APIError{
			Method:      method,
			Code:        envelope.ErrorCode,
			Description: envelope.Description,
			RetryAfter:  time.Duration(envelope.Parameters.RetryAfter) * time.Second,
		}
	}
	if result != nil {
		if err := json.Unmarshal(envelope.Result, result); err != nil {
			return fmt.Errorf("telegram %s: decodificar resultado: %w", method, err)
		}
	}
	return nil
}

// redact quita la URL (que contiene el token) de los errores de transporte y
// conserva la causa, p. ej. un timeout o una conexión rechazada.
func redact(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return fmt.Errorf("%s: %w", urlErr.Op, urlErr.Err)
	}
	return err
}
