// Package gemini es el cliente de la API de Google Gemini que genera los
// diagnósticos de ByteBalance.
//
// La llave viaja en la cabecera x-goog-api-key, nunca en la URL, así que los
// errores de transporte (que incluyen la URL) no la exponen.
package gemini

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
	"strings"
	"time"

	"bytebalance/internal/domain/insight"
)

const (
	// attemptTimeout acota cada intento; maxAttempts el número de intentos.
	attemptTimeout = 15 * time.Second
	maxAttempts    = 3
	// baseBackoff es la espera antes del primer reintento; se duplica en cada uno.
	baseBackoff = time.Second
	// maxResponseBytes limita el tamaño de las respuestas leídas (1 MiB).
	maxResponseBytes = 1 << 20
)

// APIError es un error HTTP devuelto por la API de Gemini.
type APIError struct {
	StatusCode int
	// Status es el código simbólico de Google (p. ej. RESOURCE_EXHAUSTED).
	Status  string
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("gemini: HTTP %d %s: %s", e.StatusCode, e.Status, e.Message)
}

// Temporary indica si vale la pena reintentar: límites de uso, sobrecarga o
// fallos del servidor.
func (e *APIError) Temporary() bool {
	switch e.StatusCode {
	case http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway,
		http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	return false
}

// Client genera diagnósticos con un modelo de Gemini.
type Client struct {
	baseURL string
	apiKey  string
	model   string
	http    *http.Client
	// wait espera d o hasta que se cancele ctx; se reemplaza en los tests.
	wait func(ctx context.Context, d time.Duration) error
}

// NewClient crea un cliente para baseURL (p. ej.
// https://generativelanguage.googleapis.com) que usa model con apiKey.
func NewClient(baseURL, apiKey, model string) *Client {
	return &Client{baseURL: baseURL, apiKey: apiKey, model: model, http: &http.Client{}, wait: sleep}
}

// Model devuelve el modelo con el que se generan los diagnósticos.
func (c *Client) Model() string {
	return c.model
}

// Diagnose envía el resumen al modelo y devuelve su diagnóstico tal como lo
// generó; el llamador debe sanearlo antes de mostrarlo. Reintenta con espera
// exponencial ante errores temporales.
func (c *Client) Diagnose(ctx context.Context, s insight.Snapshot) (insight.Diagnosis, error) {
	req, err := newRequest(s)
	if err != nil {
		return insight.Diagnosis{}, fmt.Errorf("gemini: %w", err)
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return insight.Diagnosis{}, fmt.Errorf("gemini: codificar petición: %w", err)
	}

	var lastErr error
	for attempt := range maxAttempts {
		if attempt > 0 {
			if err := c.wait(ctx, baseBackoff<<(attempt-1)); err != nil {
				return insight.Diagnosis{}, fmt.Errorf("gemini: %w (último error: %w)", err, lastErr)
			}
		}
		d, err := c.generate(ctx, payload)
		if err == nil || !retryable(ctx, err) {
			return d, err
		}
		slog.WarnContext(ctx, "gemini: intento fallido, se reintentará", "attempt", attempt+1, "error", err)
		lastErr = err
	}
	return insight.Diagnosis{}, lastErr
}

// retryable indica si err es temporal y el contexto del llamador sigue vigente.
func retryable(ctx context.Context, err error) bool {
	if ctx.Err() != nil {
		return false
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Temporary()
	}
	// Errores de red o el timeout de un intento.
	var urlErr *url.Error
	return errors.As(err, &urlErr) || errors.Is(err, context.DeadlineExceeded)
}

func (c *Client) generate(ctx context.Context, payload []byte) (insight.Diagnosis, error) {
	ctx, cancel := context.WithTimeout(ctx, attemptTimeout)
	defer cancel()

	endpoint := c.baseURL + "/v1beta/models/" + url.PathEscape(c.model) + ":generateContent"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return insight.Diagnosis{}, fmt.Errorf("gemini: crear petición: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return insight.Diagnosis{}, fmt.Errorf("gemini: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.WarnContext(ctx, "cerrar respuesta de Gemini", "error", err)
		}
	}()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return insight.Diagnosis{}, fmt.Errorf("gemini: leer respuesta: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return insight.Diagnosis{}, newAPIError(resp.StatusCode, raw)
	}
	return parseResponse(raw)
}

// newAPIError interpreta el cuerpo de error de Google ({"error": {...}}).
func newAPIError(statusCode int, raw []byte) *APIError {
	var body struct {
		Error struct {
			Status  string `json:"status"`
			Message string `json:"message"`
		} `json:"error"`
	}
	apiErr := &APIError{StatusCode: statusCode}
	if json.Unmarshal(raw, &body) == nil {
		apiErr.Status = body.Error.Status
		apiErr.Message = truncate(body.Error.Message, 300)
	}
	return apiErr
}

// generateResponse es la parte de la respuesta de generateContent que se usa.
type generateResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text    string `json:"text"`
				Thought bool   `json:"thought"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	PromptFeedback struct {
		BlockReason string `json:"blockReason"`
	} `json:"promptFeedback"`
}

// parseResponse extrae el diagnóstico JSON del primer candidato.
func parseResponse(raw []byte) (insight.Diagnosis, error) {
	var resp generateResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return insight.Diagnosis{}, fmt.Errorf("gemini: respuesta inválida: %w", err)
	}
	if resp.PromptFeedback.BlockReason != "" {
		return insight.Diagnosis{}, fmt.Errorf("gemini: petición bloqueada (%s)", resp.PromptFeedback.BlockReason)
	}
	if len(resp.Candidates) == 0 {
		return insight.Diagnosis{}, errors.New("gemini: respuesta sin candidatos")
	}
	candidate := resp.Candidates[0]
	if candidate.FinishReason != "STOP" {
		return insight.Diagnosis{}, fmt.Errorf("gemini: generación incompleta (%s)", candidate.FinishReason)
	}
	var text strings.Builder
	for _, part := range candidate.Content.Parts {
		if !part.Thought {
			text.WriteString(part.Text)
		}
	}

	var out diagnosisJSON
	if err := json.Unmarshal([]byte(text.String()), &out); err != nil {
		return insight.Diagnosis{}, fmt.Errorf("gemini: el diagnóstico no es JSON válido: %w", err)
	}
	return out.toDomain(), nil
}

func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func truncate(s string, max int) string {
	if r := []rune(s); len(r) > max {
		return string(r[:max]) + "…"
	}
	return s
}
