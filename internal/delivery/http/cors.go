package http

import (
	"net/http"
	"strconv"
)

// Política CORS. No se envía Access-Control-Allow-Credentials: la API se
// autentica con un token en la cabecera Authorization, no con cookies.
const (
	corsAllowedMethods = "GET, POST, PATCH, DELETE"
	corsAllowedHeaders = "Authorization, Content-Type"
	corsExposedHeaders = "Retry-After"
	corsMaxAge         = 10 * 60 // segundos que el navegador cachea el preflight
)

// cors permite que el navegador llame a la API desde los orígenes de allowed
// (p. ej. el frontend servido en otro dominio). Solo se refleja el origen si
// coincide exactamente con uno de la lista; nunca se usa el comodín "*". Un
// preflight de un origen no permitido se responde 403. Con allowed vacío no se
// añade ninguna cabecera y el navegador solo permite peticiones del mismo origen.
func cors(allowed []string, next http.Handler) http.Handler {
	if len(allowed) == 0 {
		return next
	}
	origins := make(map[string]struct{}, len(allowed))
	for _, o := range allowed {
		origins[o] = struct{}{}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}
		h := w.Header()
		h.Add("Vary", "Origin")
		_, ok := origins[origin]

		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			h.Add("Vary", "Access-Control-Request-Method")
			h.Add("Vary", "Access-Control-Request-Headers")
			if !ok {
				writeError(w, http.StatusForbidden, "origin_not_allowed", "origen no permitido")
				return
			}
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Allow-Methods", corsAllowedMethods)
			h.Set("Access-Control-Allow-Headers", corsAllowedHeaders)
			h.Set("Access-Control-Max-Age", strconv.Itoa(corsMaxAge))
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Una petición de un origen no permitido se atiende igual (no hay cookies
		// que robar), pero sin cabeceras CORS el navegador no deja leer la respuesta.
		if ok {
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Expose-Headers", corsExposedHeaders)
		}
		next.ServeHTTP(w, r)
	})
}
