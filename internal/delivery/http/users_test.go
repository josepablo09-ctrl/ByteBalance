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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestRouter(Services{Users: stubUsers{updatePreferences: tt.updateFn}})
			rec := do(h, http.MethodPatch, "/api/v1/users/me", tt.body, true)
			expectStatus(t, rec, tt.wantStatus)
		})
	}
}
