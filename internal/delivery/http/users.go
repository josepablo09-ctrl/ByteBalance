package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/user"
	"bytebalance/internal/usecase"
)

type registerRequest struct {
	Email           string `json:"email"`
	Password        string `json:"password"`
	FullName        string `json:"full_name"`
	DefaultCurrency string `json:"default_currency"`
	Timezone        string `json:"timezone"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// preferencesRequest usa punteros para distinguir "no enviado" de "vacío". La meta
// de ahorro se lee en crudo para distinguir además "no enviada" (no cambia) de
// null (la quita).
type preferencesRequest struct {
	FullName                *string         `json:"full_name"`
	DefaultCurrency         *string         `json:"default_currency"`
	Timezone                *string         `json:"timezone"`
	MonthlySavingsGoalCents json.RawMessage `json:"monthly_savings_goal_cents"`
}

// userResponse nunca incluye el hash de la contraseña ni el chat de Telegram.
type userResponse struct {
	ID              domain.ID       `json:"id"`
	Email           string          `json:"email"`
	FullName        string          `json:"full_name"`
	DefaultCurrency domain.Currency `json:"default_currency"`
	Timezone        string          `json:"timezone"`
	TelegramLinked  bool            `json:"telegram_linked"`
	// MonthlySavingsGoalCents es null si el usuario no fijó una meta.
	MonthlySavingsGoalCents *int64    `json:"monthly_savings_goal_cents"`
	CreatedAt               time.Time `json:"created_at"`
}

func newUserResponse(u *user.User) userResponse {
	return userResponse{
		ID:                      u.ID,
		Email:                   u.Email,
		FullName:                u.FullName,
		DefaultCurrency:         u.DefaultCurrency,
		Timezone:                u.Timezone,
		TelegramLinked:          u.HasTelegram(),
		MonthlySavingsGoalCents: u.MonthlySavingsGoalCents,
		CreatedAt:               u.CreatedAt,
	}
}

type tokenResponse struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// register atiende POST /api/v1/users.
func (a *api) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, r, err)
		return
	}
	if strings.TrimSpace(req.Email) == "" || req.Password == "" || strings.TrimSpace(req.FullName) == "" {
		respondError(w, r, badRequest("email, password y full_name son obligatorios"))
		return
	}

	u, err := a.Users.Register(r.Context(), usecase.RegisterInput{
		Email:           req.Email,
		Password:        req.Password,
		FullName:        req.FullName,
		DefaultCurrency: req.DefaultCurrency,
		Timezone:        req.Timezone,
	})
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, newUserResponse(u))
}

// login atiende POST /api/v1/auth/login. Los intentos se limitan por IP y por
// email; al superar el límite se responde 429 con Retry-After.
func (a *api) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, r, err)
		return
	}
	if strings.TrimSpace(req.Email) == "" || req.Password == "" {
		respondError(w, r, badRequest("email y password son obligatorios"))
		return
	}
	if ok, wait := a.loginLimits.allow(clientIP(r, a.opts.TrustedProxyHops), req.Email); !ok {
		tooManyRequests(w, r, wait, "demasiados intentos de inicio de sesión; intenta de nuevo más tarde")
		return
	}

	token, err := a.Auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, tokenResponse{AccessToken: token.Value, TokenType: "Bearer", ExpiresAt: token.ExpiresAt})
}

// profile atiende GET /api/v1/users/me.
func (a *api) profile(w http.ResponseWriter, r *http.Request) {
	u, err := a.Users.Profile(r.Context(), userIDFrom(r))
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, newUserResponse(u))
}

// updatePreferences atiende PATCH /api/v1/users/me.
func (a *api) updatePreferences(w http.ResponseWriter, r *http.Request) {
	var req preferencesRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, r, err)
		return
	}
	in := usecase.PreferencesInput{
		FullName:        req.FullName,
		DefaultCurrency: req.DefaultCurrency,
		Timezone:        req.Timezone,
	}
	if len(req.MonthlySavingsGoalCents) > 0 {
		in.SetSavingsGoal = true
		if string(req.MonthlySavingsGoalCents) != "null" {
			var goal int64
			if err := json.Unmarshal(req.MonthlySavingsGoalCents, &goal); err != nil {
				respondError(w, r, badRequest("monthly_savings_goal_cents debe ser un entero de centavos o null"))
				return
			}
			in.MonthlySavingsGoalCents = &goal
		}
	}
	if in == (usecase.PreferencesInput{}) {
		respondError(w, r, badRequest("indica al menos uno de: full_name, default_currency, timezone, monthly_savings_goal_cents"))
		return
	}

	u, err := a.Users.UpdatePreferences(r.Context(), userIDFrom(r), in)
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, newUserResponse(u))
}
