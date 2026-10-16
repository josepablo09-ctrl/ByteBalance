package http

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"bytebalance/internal/domain"
	"bytebalance/internal/usecase"
)

// respondError traduce err a una respuesta HTTP. Es el único lugar donde los
// errores de dominio se convierten en códigos de estado. Los errores no
// reconocidos se registran y se responden como 500 sin exponer detalles.
func respondError(w http.ResponseWriter, r *http.Request, err error) {
	var reqErr *requestError
	switch {
	case errors.As(err, &reqErr):
		writeError(w, http.StatusBadRequest, "bad_request", reqErr.msg)
	case errors.Is(err, usecase.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "email o contraseña incorrectos")
	case errors.Is(err, usecase.ErrTelegramNotLinked):
		writeError(w, http.StatusConflict, "telegram_not_linked", "tu cuenta no tiene Telegram vinculado")
	case errors.Is(err, usecase.ErrEmailTaken):
		writeError(w, http.StatusConflict, "email_taken", "el email ya está registrado")
	case errors.Is(err, domain.ErrInvalidInput):
		writeError(w, http.StatusUnprocessableEntity, "invalid_input", publicDetail(err, domain.ErrInvalidInput, "datos inválidos"))
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "recurso no encontrado")
	case errors.Is(err, domain.ErrConflict):
		writeError(w, http.StatusConflict, "conflict", "el recurso ya existe")
	default:
		slog.ErrorContext(r.Context(), "error interno al atender la petición",
			"error", err, "method", r.Method, "path", r.URL.Path)
		writeError(w, http.StatusInternalServerError, "internal", "error interno del servidor")
	}
}

// publicDetail extrae el mensaje que el dominio adjunta tras el error
// centinela (p. ej. "registrar usuario: entrada inválida: el email es
// obligatorio" → "el email es obligatorio"). Si no lo encuentra, o si el
// mensaje proviene de la base de datos, devuelve fallback para no exponer
// detalles internos.
func publicDetail(err, sentinel error, fallback string) string {
	msg := err.Error()
	prefix := sentinel.Error() + ": "
	i := strings.Index(msg, prefix)
	if i < 0 || strings.Contains(msg, "SQLSTATE") {
		return fallback
	}
	return msg[i+len(prefix):]
}
