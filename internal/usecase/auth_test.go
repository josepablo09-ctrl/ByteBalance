package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/user"
)

// stubVerifier acepta como válida la contraseña cuyo hash es "hashed:<contraseña>".
type stubVerifier struct {
	stubHasher
	verifyErr error
}

func (v *stubVerifier) Verify(hash, password string) (bool, error) {
	if v.verifyErr != nil {
		return false, v.verifyErr
	}
	return hash == "hashed:"+password, nil
}

// stubIssuer emite tokens predecibles y registra para quién.
type stubIssuer struct {
	err    error
	issued []domain.ID
}

func (s *stubIssuer) Issue(userID domain.ID) (string, time.Time, error) {
	if s.err != nil {
		return "", time.Time{}, s.err
	}
	s.issued = append(s.issued, userID)
	return "token-" + userID.String(), fixedNow.Add(24 * time.Hour), nil
}

func TestAuthServiceLogin(t *testing.T) {
	registered := existingUser(t, "ana@example.com", "UTC")
	registered.PasswordHash = "hashed:contraseña-segura"
	errDB := errors.New("base caída")
	errSign := errors.New("sin secreto")
	errHash := errors.New("hash corrupto")

	tests := []struct {
		name       string
		email      string
		password   string
		verifyErr  error
		issueErr   error
		wantErr    error
		wantHashed bool // si debe calcular un hash para igualar tiempos
	}{
		{name: "credenciales correctas", email: " ANA@example.com ", password: "contraseña-segura"},
		{name: "contraseña incorrecta", email: "ana@example.com", password: "otra-contraseña", wantErr: ErrInvalidCredentials},
		{name: "email inexistente", email: "nadie@example.com", password: "contraseña-segura", wantErr: ErrInvalidCredentials, wantHashed: true},
		{name: "contraseña vacía", email: "ana@example.com", password: "", wantErr: ErrInvalidCredentials},
		{name: "contraseña de más de 72 bytes", email: "ana@example.com", password: strings.Repeat("x", 73), wantErr: ErrInvalidCredentials},
		{name: "hash almacenado inválido", email: "ana@example.com", password: "contraseña-segura", verifyErr: errHash, wantErr: errHash},
		{name: "error al firmar", email: "ana@example.com", password: "contraseña-segura", issueErr: errSign, wantErr: errSign},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifier := &stubVerifier{verifyErr: tt.verifyErr}
			issuer := &stubIssuer{err: tt.issueErr}
			svc := NewAuthService(newFakeUserRepo(registered), verifier, issuer)

			token, err := svc.Login(context.Background(), tt.email, tt.password)
			if (verifier.calls > 0) != tt.wantHashed {
				t.Errorf("llamadas a Hash = %d, wantHashed %v", verifier.calls, tt.wantHashed)
			}
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Login() error = %v, se esperaba %v", err, tt.wantErr)
				}
				if token != (AccessToken{}) {
					t.Errorf("Login() con error devolvió un token: %+v", token)
				}
				return
			}
			if err != nil {
				t.Fatalf("Login() error inesperado: %v", err)
			}
			if token.Value != "token-"+registered.ID.String() || !token.ExpiresAt.Equal(fixedNow.Add(24*time.Hour)) {
				t.Errorf("Login() = %+v", token)
			}
			if len(issuer.issued) != 1 || issuer.issued[0] != registered.ID {
				t.Errorf("tokens emitidos para %v, se esperaba %v", issuer.issued, registered.ID)
			}
		})
	}

	t.Run("error de la base", func(t *testing.T) {
		svc := NewAuthService(failingUserFinder{err: errDB}, &stubVerifier{}, &stubIssuer{})
		if _, err := svc.Login(context.Background(), "ana@example.com", "contraseña-segura"); !errors.Is(err, errDB) {
			t.Fatalf("Login() error = %v, se esperaba %v", err, errDB)
		}
	})
}

// failingUserFinder simula una base de datos que falla al buscar usuarios.
type failingUserFinder struct {
	err error
}

func (f failingUserFinder) GetByEmail(context.Context, string) (*user.User, error) {
	return nil, f.err
}
