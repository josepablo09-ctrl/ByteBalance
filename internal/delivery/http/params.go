package http

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"bytebalance/internal/domain"
)

// queryTime lee un parámetro RFC 3339; si está ausente devuelve el tiempo cero.
func queryTime(q url.Values, key string) (time.Time, error) {
	raw := q.Get(key)
	if raw == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, badRequest("%s debe tener formato RFC 3339 (p. ej. 2026-10-01T00:00:00Z)", key)
	}
	return t, nil
}

// queryDate lee un parámetro con formato YYYY-MM-DD; si está ausente devuelve el tiempo cero.
func queryDate(q url.Values, key string) (time.Time, error) {
	raw := q.Get(key)
	if raw == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.DateOnly, raw)
	if err != nil {
		return time.Time{}, badRequest("%s debe tener formato YYYY-MM-DD", key)
	}
	return t, nil
}

// queryBool lee un parámetro booleano (true/false); si está ausente devuelve false.
func queryBool(q url.Values, key string) (bool, error) {
	raw := q.Get(key)
	if raw == "" {
		return false, nil
	}
	b, err := strconv.ParseBool(raw)
	if err != nil {
		return false, badRequest("%s debe ser true o false", key)
	}
	return b, nil
}

// queryInt lee un parámetro entero; si está ausente devuelve 0.
func queryInt(q url.Values, key string) (int, error) {
	raw := q.Get(key)
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, badRequest("%s debe ser un número entero", key)
	}
	return n, nil
}

// queryID lee un parámetro UUID; si está ausente devuelve nil.
func queryID(q url.Values, key string) (*domain.ID, error) {
	raw := q.Get(key)
	if raw == "" {
		return nil, nil
	}
	id, err := domain.ParseID(raw)
	if err != nil {
		return nil, badRequest("%s debe ser un UUID válido", key)
	}
	return &id, nil
}

// pathID lee un segmento de ruta UUID (p. ej. {id} en /habits/{id}/logs).
func pathID(r *http.Request, key string) (domain.ID, error) {
	id, err := domain.ParseID(r.PathValue(key))
	if err != nil {
		return domain.ID{}, badRequest("el %s de la ruta debe ser un UUID válido", key)
	}
	return id, nil
}
