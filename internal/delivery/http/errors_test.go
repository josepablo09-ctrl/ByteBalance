package http

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/finance"
	"bytebalance/internal/domain/habit"
	"bytebalance/internal/usecase"
)

func TestRespondError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantStatus  int
		wantCode    string
		wantMessage string
	}{
		{name: "petición mal formada", err: badRequest("JSON mal formado"), wantStatus: 400, wantCode: "bad_request", wantMessage: "JSON mal formado"},
		{name: "credenciales inválidas", err: usecase.ErrInvalidCredentials, wantStatus: 401, wantCode: "invalid_credentials", wantMessage: "email o contraseña incorrectos"},
		{name: "email ya registrado", err: usecase.ErrEmailTaken, wantStatus: 409, wantCode: "email_taken", wantMessage: "el email ya está registrado"},
		{
			name:        "validación de dominio con contexto",
			err:         fmt.Errorf("registrar usuario: %w: email con formato inválido", domain.ErrInvalidInput),
			wantStatus:  422,
			wantCode:    "invalid_input",
			wantMessage: "email con formato inválido",
		},
		{name: "hábito inactivo", err: fmt.Errorf("registrar cumplimiento: %w", habit.ErrInactive), wantStatus: 422, wantCode: "invalid_input", wantMessage: "el hábito está inactivo"},
		{
			name:        "detalle de la base no se expone",
			err:         fmt.Errorf("%w: ERROR: violates check constraint (SQLSTATE 23514)", domain.ErrInvalidInput),
			wantStatus:  422,
			wantCode:    "invalid_input",
			wantMessage: "datos inválidos",
		},
		{name: "no encontrado", err: fmt.Errorf("obtener hábito: %w", domain.ErrNotFound), wantStatus: 404, wantCode: "not_found", wantMessage: "recurso no encontrado"},
		{name: "conflicto genérico", err: fmt.Errorf("crear categoría: %w: duplicate key (SQLSTATE 23505)", domain.ErrConflict), wantStatus: 409, wantCode: "conflict", wantMessage: "el recurso ya existe"},
		{name: "error interno", err: errors.New("dial tcp 10.0.0.5:5432: connection refused"), wantStatus: 500, wantCode: "internal", wantMessage: "error interno del servidor"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			respondError(rec, httptest.NewRequest(http.MethodGet, "/", nil), tt.err)

			expectStatus(t, rec, tt.wantStatus)
			detail, ok := decodeJSONBody(t, rec)["error"].(map[string]any)
			if !ok {
				t.Fatalf("respuesta sin objeto error: %s", rec.Body.String())
			}
			if detail["code"] != tt.wantCode || detail["message"] != tt.wantMessage {
				t.Errorf("error = %v, se esperaba code %q y message %q", detail, tt.wantCode, tt.wantMessage)
			}
			if strings.Contains(rec.Body.String(), "SQLSTATE") || strings.Contains(rec.Body.String(), "10.0.0.5") {
				t.Errorf("la respuesta expone detalles internos: %s", rec.Body.String())
			}
		})
	}
}

func TestRecoverPanics(t *testing.T) {
	h := newTestRouter(Services{Finance: stubFinance{
		listCategories: func(domain.ID) ([]finance.Category, error) { panic("fallo inesperado") },
	}})
	rec := do(h, http.MethodGet, "/api/v1/categories", "", true)
	expectStatus(t, rec, http.StatusInternalServerError)
	if code := errorCode(t, rec); code != "internal" {
		t.Errorf("code = %q, se esperaba internal", code)
	}
}
