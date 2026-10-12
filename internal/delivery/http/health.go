// Package http contiene los handlers, routers y middlewares HTTP del backend.
package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

const healthCheckTimeout = 2 * time.Second

// Pinger verifica la disponibilidad de una dependencia (p. ej. la base de datos).
type Pinger interface {
	Ping(ctx context.Context) error
}

type healthResponse struct {
	Status string `json:"status"`
}

// Health devuelve un handler que responde 200 si db está disponible y 503 si no.
// El detalle del error solo se registra en logs, nunca se expone al cliente.
func Health(db Pinger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), healthCheckTimeout)
		defer cancel()

		status, body := http.StatusOK, healthResponse{Status: "ok"}
		if err := db.Ping(ctx); err != nil {
			slog.ErrorContext(ctx, "health check: base de datos no disponible", "error", err)
			status, body = http.StatusServiceUnavailable, healthResponse{Status: "unavailable"}
		}
		writeJSON(w, status, body)
	})
}

// writeJSON serializa body como respuesta JSON con el código status.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("escribir respuesta json", "error", err)
	}
}
