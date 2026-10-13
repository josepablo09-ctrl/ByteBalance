package http

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

const frontendOrigin = "http://localhost:5173"

func TestCORS(t *testing.T) {
	tests := []struct {
		name        string
		allowed     []string
		method      string
		origin      string
		preflight   bool
		wantStatus  int
		wantAllowed string // valor esperado de Access-Control-Allow-Origin
	}{
		{name: "origen permitido", allowed: []string{frontendOrigin}, method: http.MethodGet, origin: frontendOrigin, wantStatus: http.StatusOK, wantAllowed: frontendOrigin},
		{name: "origen no permitido", allowed: []string{frontendOrigin}, method: http.MethodGet, origin: "https://evil.example", wantStatus: http.StatusOK},
		{name: "sin cabecera Origin", allowed: []string{frontendOrigin}, method: http.MethodGet, wantStatus: http.StatusOK},
		{name: "CORS desactivado", method: http.MethodGet, origin: frontendOrigin, wantStatus: http.StatusOK},
		{name: "preflight permitido", allowed: []string{frontendOrigin}, method: http.MethodOptions, origin: frontendOrigin, preflight: true, wantStatus: http.StatusNoContent, wantAllowed: frontendOrigin},
		{name: "preflight no permitido", allowed: []string{frontendOrigin}, method: http.MethodOptions, origin: "https://evil.example", preflight: true, wantStatus: http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewRouter(withStubs(Services{}), Options{AllowedOrigins: tt.allowed})
			req := httptest.NewRequest(tt.method, "/healthz", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.preflight {
				req.Header.Set("Access-Control-Request-Method", http.MethodGet)
				req.Header.Set("Access-Control-Request-Headers", "authorization")
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			expectStatus(t, rec, tt.wantStatus)
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != tt.wantAllowed {
				t.Errorf("Access-Control-Allow-Origin = %q, se esperaba %q", got, tt.wantAllowed)
			}
			if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
				t.Errorf("no debe enviarse Access-Control-Allow-Credentials (%q)", got)
			}
			if len(tt.allowed) > 0 && tt.origin != "" && !slices.Contains(rec.Header().Values("Vary"), "Origin") {
				t.Errorf("falta Vary: Origin; cabeceras: %v", rec.Header())
			}
			if tt.wantStatus == http.StatusNoContent {
				if rec.Header().Get("Access-Control-Allow-Headers") != corsAllowedHeaders ||
					rec.Header().Get("Access-Control-Allow-Methods") != corsAllowedMethods {
					t.Errorf("preflight sin métodos o cabeceras permitidas: %v", rec.Header())
				}
			}
		})
	}
}

func TestCORSHeadersOnErrorResponses(t *testing.T) {
	// El frontend debe poder leer los errores (p. ej. 401), no solo las respuestas correctas.
	h := NewRouter(withStubs(Services{}), Options{AllowedOrigins: []string{frontendOrigin}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/balance", nil)
	req.Header.Set("Origin", frontendOrigin)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	expectStatus(t, rec, http.StatusUnauthorized)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != frontendOrigin {
		t.Errorf("Access-Control-Allow-Origin = %q en una respuesta 401", got)
	}
	if got := rec.Header().Get("Access-Control-Expose-Headers"); got != corsExposedHeaders {
		t.Errorf("Access-Control-Expose-Headers = %q", got)
	}
}
