package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

// maxBodyBytes limita el tamaño de los cuerpos JSON aceptados (1 MiB).
const maxBodyBytes = 1 << 20

// errorResponse es el formato de todas las respuestas de error de la API.
type errorResponse struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// writeJSON serializa body como respuesta JSON con el código status.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("escribir respuesta json", "error", err)
	}
}

// writeError responde con el formato de error estándar.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorResponse{Error: errorDetail{Code: code, Message: message}})
}

// requestError es un error en la forma de la petición (JSON mal formado,
// parámetro con formato inválido, campo obligatorio ausente). Se responde con 400.
type requestError struct {
	msg string
}

func (e *requestError) Error() string { return e.msg }

func badRequest(format string, args ...any) error {
	return &requestError{msg: fmt.Sprintf(format, args...)}
}

// decodeJSON lee del cuerpo exactamente un objeto JSON en dst. Rechaza cuerpos
// vacíos, campos desconocidos y cuerpos de más de maxBodyBytes.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	err := decode(w, r, dst)
	if errors.Is(err, io.EOF) {
		return badRequest("el cuerpo de la petición no puede estar vacío")
	}
	return err
}

// decodeOptionalJSON es como decodeJSON, pero acepta un cuerpo vacío y deja dst sin cambios.
func decodeOptionalJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	err := decode(w, r, dst)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

func decode(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		var syntaxErr *json.SyntaxError
		var typeErr *json.UnmarshalTypeError
		switch {
		case errors.Is(err, io.EOF):
			return io.EOF
		case errors.As(err, &maxErr):
			return badRequest("el cuerpo de la petición supera %d bytes", maxBodyBytes)
		case errors.As(err, &syntaxErr), errors.Is(err, io.ErrUnexpectedEOF):
			return badRequest("JSON mal formado")
		case errors.As(err, &typeErr):
			return badRequest("el campo %q tiene un tipo inválido", typeErr.Field)
		default:
			// Campos desconocidos o valores que no se pueden interpretar (p. ej. un UUID inválido).
			return badRequest("cuerpo inválido: %v", err)
		}
	}
	if dec.More() {
		return badRequest("el cuerpo debe contener un único objeto JSON")
	}
	return nil
}
