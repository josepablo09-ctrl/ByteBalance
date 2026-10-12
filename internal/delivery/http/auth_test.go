package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/finance"
)

func TestRequireAuth(t *testing.T) {
	tests := []struct {
		name          string
		authorization string
		wantStatus    int
	}{
		{name: "sin cabecera", wantStatus: http.StatusUnauthorized},
		{name: "esquema Basic", authorization: "Basic dXNlcjpwYXNz", wantStatus: http.StatusUnauthorized},
		{name: "Bearer sin token", authorization: "Bearer ", wantStatus: http.StatusUnauthorized},
		{name: "token inválido", authorization: "Bearer otro-token", wantStatus: http.StatusUnauthorized},
		{name: "token válido", authorization: "Bearer " + validToken, wantStatus: http.StatusOK},
		{name: "esquema sin distinguir mayúsculas", authorization: "bearer " + validToken, wantStatus: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotUser domain.ID
			h := newTestRouter(Services{Finance: stubFinance{
				listCategories: func(userID domain.ID) ([]finance.Category, error) {
					gotUser = userID
					return nil, nil
				},
			}})
			req := httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil)
			if tt.authorization != "" {
				req.Header.Set("Authorization", tt.authorization)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			expectStatus(t, rec, tt.wantStatus)
			if tt.wantStatus == http.StatusUnauthorized {
				if rec.Header().Get("WWW-Authenticate") == "" {
					t.Error("una respuesta 401 debe incluir WWW-Authenticate")
				}
				if code := errorCode(t, rec); code != "unauthorized" {
					t.Errorf("code = %q, se esperaba unauthorized", code)
				}
				if !gotUser.IsZero() {
					t.Error("el handler protegido no debe ejecutarse sin autenticación")
				}
				return
			}
			if gotUser != testUserID {
				t.Errorf("userID en el handler = %v, se esperaba %v", gotUser, testUserID)
			}
		})
	}
}

func TestPublicRoutesDoNotRequireAuth(t *testing.T) {
	h := newTestRouter(Services{})
	for _, path := range []string{"/api/v1/users", "/api/v1/auth/login"} {
		rec := do(h, http.MethodPost, path, `{}`, false)
		if rec.Code == http.StatusUnauthorized {
			t.Errorf("POST %s exige autenticación y debería ser pública", path)
		}
	}
	if rec := do(h, http.MethodGet, "/healthz", "", false); rec.Code != http.StatusOK {
		t.Errorf("GET /healthz = %d, se esperaba 200 sin autenticación", rec.Code)
	}
}
