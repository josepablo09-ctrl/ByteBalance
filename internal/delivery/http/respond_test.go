package http

import (
	"net/http"
	"strings"
	"testing"
)

// TestDecodeJSON verifica la lectura estricta del cuerpo a través de un endpoint real.
func TestDecodeJSON(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "cuerpo vacío", body: ""},
		{name: "JSON mal formado", body: `{"email": `},
		{name: "campo desconocido", body: `{"email":"a@b.com","password":"x","full_name":"A","is_admin":true}`},
		{name: "tipo inválido", body: `{"email": 123}`},
		{name: "dos objetos", body: `{"email":"a@b.com"} {"email":"c@d.com"}`},
		{name: "arreglo en lugar de objeto", body: `[]`},
		{name: "cuerpo demasiado grande", body: `{"full_name":"` + strings.Repeat("a", maxBodyBytes) + `"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(newTestRouter(Services{}), http.MethodPost, "/api/v1/users", tt.body, false)
			expectStatus(t, rec, http.StatusBadRequest)
			if code := errorCode(t, rec); code != "bad_request" {
				t.Errorf("code = %q, se esperaba bad_request", code)
			}
		})
	}
}
