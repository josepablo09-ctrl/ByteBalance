package http

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/user"
	"bytebalance/internal/usecase"
)

func sampleUser() *user.User {
	return &user.User{
		ID:              testUserID,
		Email:           "ana@example.com",
		PasswordHash:    "$2a$12$hash-que-nunca-debe-salir",
		FullName:        "Ana Pérez",
		DefaultCurrency: "CRC",
		Timezone:        "America/Costa_Rica",
		CreatedAt:       time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC),
	}
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		registerFn func(usecase.RegisterInput) (*user.User, error)
		wantStatus int
		wantCode   string
	}{
		{
			name: "registro correcto",
			body: `{"email":"ana@example.com","password":"contraseña-segura","full_name":"Ana Pérez","default_currency":"CRC","timezone":"America/Costa_Rica"}`,
			registerFn: func(in usecase.RegisterInput) (*user.User, error) {
				if in.Email != "ana@example.com" || in.Password != "contraseña-segura" || in.DefaultCurrency != "CRC" {
					t.Errorf("RegisterInput = %+v", in)
				}
				return sampleUser(), nil
			},
			wantStatus: http.StatusCreated,
		},
		{name: "falta password", body: `{"email":"ana@example.com","full_name":"Ana"}`, wantStatus: http.StatusBadRequest, wantCode: "bad_request"},
		{name: "nombre en blanco", body: `{"email":"ana@example.com","password":"x","full_name":"  "}`, wantStatus: http.StatusBadRequest, wantCode: "bad_request"},
		{
			name:       "email ya registrado",
			body:       `{"email":"ana@example.com","password":"contraseña-segura","full_name":"Ana"}`,
			registerFn: func(usecase.RegisterInput) (*user.User, error) { return nil, usecase.ErrEmailTaken },
			wantStatus: http.StatusConflict,
			wantCode:   "email_taken",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestRouter(Services{Users: stubUsers{register: tt.registerFn}})
			rec := do(h, http.MethodPost, "/api/v1/users", tt.body, false)

			expectStatus(t, rec, tt.wantStatus)
			if tt.wantCode != "" {
				if code := errorCode(t, rec); code != tt.wantCode {
					t.Errorf("code = %q, se esperaba %q", code, tt.wantCode)
				}
				return
			}
			body := decodeJSONBody(t, rec)
			if body["id"] != testUserID.String() || body["email"] != "ana@example.com" || body["telegram_linked"] != false {
				t.Errorf("respuesta = %v", body)
			}
			if strings.Contains(rec.Body.String(), "hash") || strings.Contains(rec.Body.String(), "password") {
				t.Errorf("la respuesta expone datos de la contraseña: %s", rec.Body.String())
			}
		})
	}
}

func TestLogin(t *testing.T) {
	expiresAt := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		body       string
		loginFn    func(string, string) (usecase.AccessToken, error)
		wantStatus int
		wantCode   string
	}{
		{
			name: "credenciales correctas",
			body: `{"email":"ana@example.com","password":"contraseña-segura"}`,
			loginFn: func(email, password string) (usecase.AccessToken, error) {
				return usecase.AccessToken{Value: "jwt-firmado", ExpiresAt: expiresAt}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "credenciales incorrectas",
			body: `{"email":"ana@example.com","password":"otra"}`,
			loginFn: func(string, string) (usecase.AccessToken, error) {
				return usecase.AccessToken{}, usecase.ErrInvalidCredentials
			},
			wantStatus: http.StatusUnauthorized,
			wantCode:   "invalid_credentials",
		},
		{name: "falta email", body: `{"password":"x"}`, wantStatus: http.StatusBadRequest, wantCode: "bad_request"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestRouter(Services{Auth: stubAuth{login: tt.loginFn}})
			rec := do(h, http.MethodPost, "/api/v1/auth/login", tt.body, false)

			expectStatus(t, rec, tt.wantStatus)
			if tt.wantCode != "" {
				if code := errorCode(t, rec); code != tt.wantCode {
					t.Errorf("code = %q, se esperaba %q", code, tt.wantCode)
				}
				return
			}
			body := decodeJSONBody(t, rec)
			if body["access_token"] != "jwt-firmado" || body["token_type"] != "Bearer" || body["expires_at"] != "2026-10-12T12:00:00Z" {
				t.Errorf("respuesta = %v", body)
			}
		})
	}
}

func TestUpdatePreferences(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		updateFn   func(domain.ID, usecase.PreferencesInput) (*user.User, error)
		wantStatus int
	}{
		{
			name: "cambio parcial",
			body: `{"timezone":"Europe/Madrid"}`,
			updateFn: func(id domain.ID, in usecase.PreferencesInput) (*user.User, error) {
				if id != testUserID {
					t.Errorf("userID = %v, se esperaba el del token", id)
				}
				if in.Timezone == nil || *in.Timezone != "Europe/Madrid" || in.FullName != nil || in.DefaultCurrency != nil {
					t.Errorf("PreferencesInput = %+v", in)
				}
				u := sampleUser()
				u.Timezone = *in.Timezone
				return u, nil
			},
			wantStatus: http.StatusOK,
		},
		{name: "sin campos", body: `{}`, wantStatus: http.StatusBadRequest},
		{name: "solo nulos", body: `{"timezone":null}`, wantStatus: http.StatusBadRequest},
		{
			name: "fija la meta de ahorro",
			body: `{"monthly_savings_goal_cents":10000000}`,
			updateFn: func(_ domain.ID, in usecase.PreferencesInput) (*user.User, error) {
				if !in.SetSavingsGoal || in.MonthlySavingsGoalCents == nil || *in.MonthlySavingsGoalCents != 10000000 || in.Timezone != nil {
					t.Errorf("PreferencesInput = %+v", in)
				}
				return sampleUser(), nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "null quita la meta de ahorro",
			body: `{"monthly_savings_goal_cents":null}`,
			updateFn: func(_ domain.ID, in usecase.PreferencesInput) (*user.User, error) {
				if !in.SetSavingsGoal || in.MonthlySavingsGoalCents != nil {
					t.Errorf("PreferencesInput = %+v", in)
				}
				return sampleUser(), nil
			},
			wantStatus: http.StatusOK,
		},
		{name: "meta que no es entero", body: `{"monthly_savings_goal_cents":"mucho"}`, wantStatus: http.StatusBadRequest},
		{name: "meta con decimales", body: `{"monthly_savings_goal_cents":10.5}`, wantStatus: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestRouter(Services{Users: stubUsers{updatePreferences: tt.updateFn}})
			rec := do(h, http.MethodPatch, "/api/v1/users/me", tt.body, true)
			expectStatus(t, rec, tt.wantStatus)
		})
	}
}

func TestProfile(t *testing.T) {
	tests := []struct {
		name          string
		authenticated bool
		profileFn     func(domain.ID) (*user.User, error)
		wantStatus    int
	}{
		{
			name:          "usuario del token",
			authenticated: true,
			profileFn: func(id domain.ID) (*user.User, error) {
				if id != testUserID {
					t.Errorf("userID = %v, se esperaba el del token", id)
				}
				return sampleUser(), nil
			},
			wantStatus: http.StatusOK,
		},
		{name: "sin token", wantStatus: http.StatusUnauthorized},
		{
			name:          "usuario eliminado",
			authenticated: true,
			profileFn:     func(domain.ID) (*user.User, error) { return nil, domain.ErrNotFound },
			wantStatus:    http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestRouter(Services{Users: stubUsers{profile: tt.profileFn}})
			rec := do(h, http.MethodGet, "/api/v1/users/me", "", tt.authenticated)
			expectStatus(t, rec, tt.wantStatus)
			if tt.wantStatus != http.StatusOK {
				return
			}
			body := decodeJSONBody(t, rec)
			if body["id"] != testUserID.String() || body["email"] != "ana@example.com" || body["timezone"] != "America/Costa_Rica" {
				t.Errorf("respuesta = %v", body)
			}
			// Sin meta, el campo se envía como null (no se omite).
			if goal, ok := body["monthly_savings_goal_cents"]; !ok || goal != nil {
				t.Errorf("monthly_savings_goal_cents = %v (presente %v), se esperaba null", goal, ok)
			}
			if strings.Contains(rec.Body.String(), "hash") {
				t.Errorf("la respuesta expone el hash de la contraseña: %s", rec.Body.String())
			}
		})
	}
}
