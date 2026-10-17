package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	want := map[string]string{
		"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
		"Referrer-Policy":         "no-referrer",
		"Cache-Control":           "no-store",
	}
	requests := []struct {
		name          string
		method, path  string
		authenticated bool
		wantStatus    int
	}{
		{name: "health", method: http.MethodGet, path: "/healthz", wantStatus: http.StatusOK},
		{name: "error 401", method: http.MethodGet, path: "/api/v1/users/me", wantStatus: http.StatusUnauthorized},
		{name: "ruta inexistente", method: http.MethodGet, path: "/no-existe", wantStatus: http.StatusNotFound},
	}
	for _, hsts := range []bool{false, true} {
		h := NewRouter(withStubs(Services{}), Options{HSTS: hsts})
		for _, tt := range requests {
			t.Run(tt.name, func(t *testing.T) {
				rec := do(h, tt.method, tt.path, "", tt.authenticated)
				expectStatus(t, rec, tt.wantStatus)
				for name, value := range want {
					if got := rec.Header().Get(name); got != value {
						t.Errorf("%s = %q, se esperaba %q", name, got, value)
					}
				}
				gotHSTS := rec.Header().Get("Strict-Transport-Security")
				if hsts && gotHSTS != hstsValue {
					t.Errorf("Strict-Transport-Security = %q, se esperaba %q", gotHSTS, hstsValue)
				}
				if !hsts && gotHSTS != "" {
					t.Errorf("en desarrollo no debe enviarse HSTS, se envió %q", gotHSTS)
				}
			})
		}
	}
}

func TestSecurityHeadersKeepCORS(t *testing.T) {
	h := NewRouter(withStubs(Services{}), Options{AllowedOrigins: []string{"https://app.bytebalance.com"}, HSTS: true})
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/users/me", nil)
	req.Header.Set("Origin", "https://app.bytebalance.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	expectStatus(t, rec, http.StatusNoContent)
	if rec.Header().Get("Access-Control-Allow-Origin") != "https://app.bytebalance.com" || rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("cabeceras del preflight = %v", rec.Header())
	}
}
