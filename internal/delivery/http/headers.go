package http

import "net/http"

// hstsValue pide al navegador usar solo HTTPS con la API durante un año.
const hstsValue = "max-age=31536000; includeSubDomains"

// securityHeaders añade cabeceras de seguridad a todas las respuestas. La API
// solo devuelve JSON, así que su CSP no permite cargar ni incrustar nada.
// Cache-Control: no-store evita que proxies o el navegador guarden datos
// financieros. Con hsts se envía Strict-Transport-Security, que solo debe
// activarse cuando la API se sirve por HTTPS (fuera de desarrollo).
func securityHeaders(hsts bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cache-Control", "no-store")
		if hsts {
			h.Set("Strict-Transport-Security", hstsValue)
		}
		next.ServeHTTP(w, r)
	})
}
