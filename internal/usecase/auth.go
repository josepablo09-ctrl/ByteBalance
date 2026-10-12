package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/user"
)

// ErrInvalidCredentials indica que el email o la contraseña no son correctos.
// No distingue cuál de los dos falló para no revelar qué emails existen.
var ErrInvalidCredentials = errors.New("credenciales inválidas")

// PasswordVerifier comprueba contraseñas contra su hash.
type PasswordVerifier interface {
	PasswordHasher
	// Verify indica si password corresponde a hash.
	Verify(hash, password string) (bool, error)
}

// TokenIssuer emite tokens de acceso para un usuario autenticado.
type TokenIssuer interface {
	Issue(userID domain.ID) (token string, expiresAt time.Time, err error)
}

type userByEmail interface {
	GetByEmail(ctx context.Context, email string) (*user.User, error)
}

// AccessToken es un token de acceso emitido tras un login correcto.
type AccessToken struct {
	Value     string
	ExpiresAt time.Time
}

// AuthService implementa el caso de uso de inicio de sesión.
type AuthService struct {
	users     userByEmail
	passwords PasswordVerifier
	tokens    TokenIssuer
}

// NewAuthService crea un AuthService.
func NewAuthService(users userByEmail, passwords PasswordVerifier, tokens TokenIssuer) *AuthService {
	return &AuthService{users: users, passwords: passwords, tokens: tokens}
}

// Login valida las credenciales y emite un token de acceso. Devuelve
// ErrInvalidCredentials si el email no existe o la contraseña no coincide.
func (s *AuthService) Login(ctx context.Context, email, password string) (AccessToken, error) {
	// Ninguna contraseña registrada puede estar vacía ni superar el máximo.
	if password == "" || len(password) > maxPasswordBytes {
		return AccessToken{}, ErrInvalidCredentials
	}

	u, err := s.users.GetByEmail(ctx, strings.TrimSpace(email))
	if errors.Is(err, domain.ErrNotFound) {
		// Se calcula un hash igualmente para que el tiempo de respuesta no
		// revele si el email está registrado.
		if _, err := s.passwords.Hash(password); err != nil {
			return AccessToken{}, fmt.Errorf("iniciar sesión: %w", err)
		}
		return AccessToken{}, ErrInvalidCredentials
	}
	if err != nil {
		return AccessToken{}, fmt.Errorf("iniciar sesión: %w", err)
	}

	ok, err := s.passwords.Verify(u.PasswordHash, password)
	if err != nil {
		return AccessToken{}, fmt.Errorf("iniciar sesión: %w", err)
	}
	if !ok {
		return AccessToken{}, ErrInvalidCredentials
	}

	token, expiresAt, err := s.tokens.Issue(u.ID)
	if err != nil {
		return AccessToken{}, fmt.Errorf("iniciar sesión: %w", err)
	}
	return AccessToken{Value: token, ExpiresAt: expiresAt}, nil
}
