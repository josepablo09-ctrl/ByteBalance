package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// token es ficticio, con el formato de BotFather.
const token = "123456789:AAFicticioFicticioFicticioFicticio12"

// fakeAPI simula la Bot API: registra la última petición y responde reply.
func fakeAPI(t *testing.T, reply string) (*Client, *http.Request, *map[string]any) {
	t.Helper()
	var lastReq http.Request
	lastBody := map[string]any{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastReq = *r
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("leer cuerpo: %v", err)
		}
		if err := json.Unmarshal(raw, &lastBody); err != nil {
			t.Errorf("cuerpo no es JSON: %s", raw)
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, reply); err != nil {
			t.Errorf("escribir respuesta: %v", err)
		}
	}))
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, token), &lastReq, &lastBody
}

func TestGetMe(t *testing.T) {
	client, req, _ := fakeAPI(t, `{"ok":true,"result":{"id":42,"username":"ByteBalanceBot"}}`)
	bot, err := client.GetMe(context.Background())
	if err != nil {
		t.Fatalf("GetMe() error inesperado: %v", err)
	}
	if bot.Username != "ByteBalanceBot" || bot.ID != 42 {
		t.Errorf("bot = %+v", bot)
	}
	if req.URL.Path != "/bot"+token+"/getMe" || req.Method != http.MethodPost {
		t.Errorf("petición = %s %s", req.Method, req.URL.Path)
	}
}

func TestSendMessage(t *testing.T) {
	client, req, body := fakeAPI(t, `{"ok":true,"result":{}}`)
	if err := client.SendMessage(context.Background(), 777, "Hola"); err != nil {
		t.Fatalf("SendMessage() error inesperado: %v", err)
	}
	if !strings.HasSuffix(req.URL.Path, "/sendMessage") || (*body)["chat_id"] != float64(777) || (*body)["text"] != "Hola" {
		t.Errorf("petición = %s %v", req.URL.Path, *body)
	}
}

func TestGetUpdates(t *testing.T) {
	client, _, body := fakeAPI(t, `{"ok":true,"result":[
		{"update_id":10,"message":{"chat":{"id":777,"type":"private"},"text":"/start abc"}},
		{"update_id":11}
	]}`)
	updates, err := client.GetUpdates(context.Background(), 10, 30*time.Second)
	if err != nil {
		t.Fatalf("GetUpdates() error inesperado: %v", err)
	}
	if len(updates) != 2 || updates[0].Message.Text != "/start abc" || updates[0].Message.Chat.Type != "private" || updates[1].Message != nil {
		t.Errorf("updates = %+v", updates)
	}
	if (*body)["offset"] != float64(10) || (*body)["timeout"] != float64(30) {
		t.Errorf("cuerpo = %v", *body)
	}
}

func TestAPIError(t *testing.T) {
	client, _, _ := fakeAPI(t, `{"ok":false,"error_code":429,"description":"Too Many Requests","parameters":{"retry_after":7}}`)
	err := client.SendMessage(context.Background(), 1, "x")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, se esperaba *APIError", err)
	}
	if apiErr.Code != 429 || apiErr.RetryAfter != 7*time.Second || apiErr.Method != "sendMessage" {
		t.Errorf("APIError = %+v", apiErr)
	}
}

func TestErrorsNeverContainToken(t *testing.T) {
	tests := []struct {
		name   string
		client func(t *testing.T) *Client
	}{
		{
			name: "servidor caído",
			client: func(*testing.T) *Client {
				srv := httptest.NewServer(http.NotFoundHandler())
				srv.Close() // La conexión será rechazada.
				return NewClient(srv.URL, token)
			},
		},
		{
			name: "respuesta que no es JSON",
			client: func(t *testing.T) *Client {
				c, _, _ := fakeAPI(t, `<html>502</html>`)
				return c
			},
		},
		{
			name: "error de la API",
			client: func(t *testing.T) *Client {
				c, _, _ := fakeAPI(t, `{"ok":false,"error_code":401,"description":"Unauthorized"}`)
				return c
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.client(t).GetMe(context.Background())
			if err == nil {
				t.Fatal("se esperaba un error")
			}
			if strings.Contains(err.Error(), token) || strings.Contains(err.Error(), "AAFicticio") {
				t.Fatalf("el error expone el token: %v", err)
			}
		})
	}
}
