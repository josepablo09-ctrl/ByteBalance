package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"bytebalance/internal/domain/finance"
	"bytebalance/internal/domain/habit"
	"bytebalance/internal/domain/insight"
)

// testKey es una llave ficticia.
const testKey = "llave-de-prueba-ficticia-0000000000"

var sampleSnapshot = insight.Snapshot{
	From: time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC),
	To:   time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC),
	Finances: []insight.CurrencySummary{{
		Currency: "CRC", IncomeCents: 600_000_00, ExpenseCents: 450_000_50, SavingsCents: 80_000_00, Transactions: 12,
		TopExpenses: []insight.CategoryAmount{{Name: "Comida", Cents: 200_000_00}},
	}},
	Habits:      []insight.HabitSummary{{Name: "Meditar", Frequency: habit.Daily, TargetCount: 1, PeriodsMet: 12, PeriodsTotal: 29, CurrentStreak: 3}},
	SavingsGoal: &finance.Money{Cents: 100_000_00, Currency: "CRC"},
}

const diagnosisText = `{"headline":"Ahorras el 25 %","summary":"Vas bien.","finance_insights":["Comida es tu mayor gasto"],"habit_insights":["Meditar: 12 de 29 días"],"suggestions":[{"title":"Tope","detail":"Fija 180 000 CRC para comida"}]}`

// okBody es una respuesta de generateContent con el diagnóstico de ejemplo,
// precedido de una parte de razonamiento que debe ignorarse.
func okBody(t *testing.T, text string) string {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"candidates": []map[string]any{{
			"content": map[string]any{"role": "model", "parts": []map[string]any{
				{"text": "pensando…", "thought": true},
				{"text": text},
			}},
			"finishReason": "STOP",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

// reply responde con status y body desde un handler del servidor simulado.
func reply(t *testing.T, w http.ResponseWriter, status int, body string) {
	t.Helper()
	w.WriteHeader(status)
	if _, err := io.WriteString(w, body); err != nil {
		t.Errorf("escribir respuesta: %v", err)
	}
}

// newTestClient apunta a un servidor simulado y no espera entre reintentos.
func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := NewClient(srv.URL, testKey, "gemini-test")
	c.wait = func(ctx context.Context, _ time.Duration) error { return ctx.Err() }
	return c
}

func TestDiagnose(t *testing.T) {
	var got struct {
		path, key, query string
		body             request
	}
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got.path, got.key, got.query = r.URL.Path, r.Header.Get("x-goog-api-key"), r.URL.RawQuery
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("leer petición: %v", err)
		}
		if err := json.Unmarshal(raw, &got.body); err != nil {
			t.Errorf("petición inválida: %v", err)
		}
		reply(t, w, http.StatusOK, okBody(t, diagnosisText))
	})

	d, err := c.Diagnose(context.Background(), sampleSnapshot)
	if err != nil {
		t.Fatalf("Diagnose() error inesperado: %v", err)
	}

	if got.path != "/v1beta/models/gemini-test:generateContent" {
		t.Errorf("ruta = %q", got.path)
	}
	if got.key != testKey || strings.Contains(got.query, testKey) {
		t.Errorf("la llave debe ir solo en la cabecera (cabecera %q, query %q)", got.key, got.query)
	}
	cfg := got.body.GenerationConfig
	if cfg.ResponseMIMEType != "application/json" || cfg.ResponseSchema == nil || cfg.ThinkingConfig.ThinkingLevel != "low" {
		t.Errorf("generationConfig = %+v", cfg)
	}
	prompt := got.body.Contents[0].Parts[0].Text
	for _, want := range []string{`"moneda":"CRC"`, `"gastos":"450 000,50"`, `"excedente_pct":25`, `"apartado_al_ahorro":"80 000"`, `"meta_de_ahorro_mensual":{"moneda":"CRC","monto":"100 000"}`, `"categoría":"Comida"`, `"hábito":"Meditar"`, `"frecuencia":"diaria"`, `"desde":"2026-09-17"`} {
		if !strings.Contains(prompt, want) {
			t.Errorf("el prompt no contiene %s:\n%s", want, prompt)
		}
	}

	want := insight.Diagnosis{
		Headline: "Ahorras el 25 %", Summary: "Vas bien.",
		FinanceInsights: []string{"Comida es tu mayor gasto"}, HabitInsights: []string{"Meditar: 12 de 29 días"},
		Suggestions: []insight.Suggestion{{Title: "Tope", Detail: "Fija 180 000 CRC para comida"}},
	}
	if d.Headline != want.Headline || d.Summary != want.Summary || len(d.Suggestions) != 1 || d.Suggestions[0] != want.Suggestions[0] ||
		d.FinanceInsights[0] != want.FinanceInsights[0] || d.HabitInsights[0] != want.HabitInsights[0] {
		t.Errorf("Diagnose() = %+v, se esperaba %+v", d, want)
	}
}

func TestDiagnoseRetriesTemporaryErrors(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			reply(t, w, http.StatusServiceUnavailable, `{"error":{"code":503,"status":"UNAVAILABLE","message":"high demand"}}`)
			return
		}
		reply(t, w, http.StatusOK, okBody(t, diagnosisText))
	})
	if _, err := c.Diagnose(context.Background(), sampleSnapshot); err != nil {
		t.Fatalf("Diagnose() error inesperado: %v", err)
	}
	if calls.Load() != 3 {
		t.Errorf("intentos = %d, se esperaban 3", calls.Load())
	}
}

func TestDiagnoseErrors(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		wantCalls int32
		check     func(t *testing.T, err error)
	}{
		{
			name: "límite de uso persistente: agota los intentos", status: http.StatusTooManyRequests,
			body:      `{"error":{"code":429,"status":"RESOURCE_EXHAUSTED","message":"quota"}}`,
			wantCalls: maxAttempts,
			check: func(t *testing.T, err error) {
				var apiErr *APIError
				if !errors.As(err, &apiErr) || apiErr.Status != "RESOURCE_EXHAUSTED" {
					t.Errorf("error = %v, se esperaba APIError RESOURCE_EXHAUSTED", err)
				}
			},
		},
		{
			name: "llave inválida: no reintenta", status: http.StatusBadRequest,
			body:      `{"error":{"code":400,"status":"INVALID_ARGUMENT","message":"API key not valid"}}`,
			wantCalls: 1,
		},
		{name: "respuesta bloqueada", status: http.StatusOK, body: `{"promptFeedback":{"blockReason":"SAFETY"}}`, wantCalls: 1},
		{name: "sin candidatos", status: http.StatusOK, body: `{"candidates":[]}`, wantCalls: 1},
		{
			name: "respuesta cortada", status: http.StatusOK, wantCalls: 1,
			body: `{"candidates":[{"content":{"parts":[{"text":"{\"headline\":"}]},"finishReason":"MAX_TOKENS"}]}`,
		},
		{name: "texto que no es JSON", status: http.StatusOK, body: okBody(t, "Hola, aquí va tu análisis"), wantCalls: 1},
		{name: "cuerpo que no es JSON", status: http.StatusOK, body: "<html>", wantCalls: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int32
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				reply(t, w, tt.status, tt.body)
			})
			_, err := c.Diagnose(context.Background(), sampleSnapshot)
			if err == nil {
				t.Fatal("Diagnose() se esperaba error")
			}
			if strings.Contains(err.Error(), testKey) {
				t.Errorf("el error expone la llave: %v", err)
			}
			if calls.Load() != tt.wantCalls {
				t.Errorf("intentos = %d, se esperaban %d", calls.Load(), tt.wantCalls)
			}
			if tt.check != nil {
				tt.check(t, err)
			}
		})
	}
}

func TestDiagnoseStopsWhenContextEnds(t *testing.T) {
	var calls atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		cancel() // el usuario abandona la petición durante el primer intento
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	if _, err := c.Diagnose(ctx, sampleSnapshot); err == nil {
		t.Fatal("Diagnose() se esperaba error")
	}
	if calls.Load() != 1 {
		t.Errorf("intentos = %d, no debe reintentar con el contexto cancelado", calls.Load())
	}
}

func TestDiagnoseNetworkErrorDoesNotLeakKey(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", testKey, "gemini-test")
	c.wait = func(ctx context.Context, _ time.Duration) error { return ctx.Err() }
	_, err := c.Diagnose(context.Background(), sampleSnapshot)
	if err == nil {
		t.Fatal("Diagnose() se esperaba error de conexión")
	}
	if strings.Contains(err.Error(), testKey) {
		t.Errorf("el error expone la llave: %v", err)
	}
}

func TestFormatAmount(t *testing.T) {
	tests := map[int64]string{
		0: "0", 5: "0,05", 100: "1", 123456: "1 234,56", -250: "-2,50",
		95_000_00: "95 000", 750_000_00: "750 000", 1_234_567_89: "1 234 567,89", -100_000_00: "-100 000",
	}
	for cents, want := range tests {
		if got := formatAmount(cents); got != want {
			t.Errorf("formatAmount(%d) = %q, se esperaba %q", cents, got, want)
		}
	}
}

func TestPercent(t *testing.T) {
	value := func(p *int) any {
		if p == nil {
			return nil
		}
		return *p
	}
	tests := []struct {
		part, whole int64
		want        any
	}{
		{475, 750, 63},
		{155, 120, 129},
		{-50, 100, -50},
		{10, 0, nil},
		{10, -5, nil},
	}
	for _, tt := range tests {
		if got := value(percent(tt.part, tt.whole)); got != tt.want {
			t.Errorf("percent(%d, %d) = %v, se esperaba %v", tt.part, tt.whole, got, tt.want)
		}
	}
}

func TestPromptDataHasNoIdentifiers(t *testing.T) {
	data, err := json.Marshal(newPromptData(insight.Snapshot{}))
	if err != nil {
		t.Fatal(err)
	}
	// Sin datos, las listas deben ser [] (no null) para que el modelo lo entienda.
	if !strings.Contains(string(data), `"finanzas":[]`) || !strings.Contains(string(data), `"hábitos":[]`) {
		t.Errorf("promptData vacío = %s", data)
	}
	for _, field := range []string{"id", "email", "user", "nombre"} {
		if strings.Contains(strings.ToLower(string(data)), `"`+field) {
			t.Errorf("el prompt incluye el campo %q", field)
		}
	}
}
