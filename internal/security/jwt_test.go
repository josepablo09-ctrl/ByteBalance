package security

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"bytebalance/internal/domain"
)

var testSecret = []byte(strings.Repeat("s", MinJWTSecretLen))

func TestNewJWT(t *testing.T) {
	tests := []struct {
		name    string
		secret  []byte
		ttl     time.Duration
		wantErr bool
	}{
		{name: "válido", secret: testSecret, ttl: time.Hour},
		{name: "secreto corto", secret: []byte("corto"), ttl: time.Hour, wantErr: true},
		{name: "ttl cero", secret: testSecret, ttl: 0, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewJWT(tt.secret, tt.ttl, nil)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewJWT() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJWTIssueAndVerify(t *testing.T) {
	issuedAt := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	clock := issuedAt
	j, err := NewJWT(testSecret, 24*time.Hour, func() time.Time { return clock })
	if err != nil {
		t.Fatalf("NewJWT: %v", err)
	}
	userID := domain.NewID()
	token, expiresAt, err := j.Issue(userID)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if !expiresAt.Equal(issuedAt.Add(24 * time.Hour)) {
		t.Errorf("expiresAt = %v, se esperaba 24 h después de emitir", expiresAt)
	}

	other, err := NewJWT([]byte(strings.Repeat("x", MinJWTSecretLen)), time.Hour, func() time.Time { return issuedAt })
	if err != nil {
		t.Fatalf("NewJWT: %v", err)
	}
	foreignToken, _, err := other.Issue(userID)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	sign := func(method jwt.SigningMethod, key any, claims jwt.Claims) string {
		t.Helper()
		s, err := jwt.NewWithClaims(method, claims).SignedString(key)
		if err != nil {
			t.Fatalf("firmar token de prueba: %v", err)
		}
		return s
	}
	validClaims := func() jwt.RegisteredClaims {
		return jwt.RegisteredClaims{
			Issuer:    jwtIssuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(issuedAt.Add(time.Hour)),
		}
	}
	withClaims := func(modify func(c *jwt.RegisteredClaims)) jwt.RegisteredClaims {
		c := validClaims()
		modify(&c)
		return c
	}

	// tamper reemplaza el payload de token por el de donor, conservando la firma original.
	tamper := func(token, donor string) string {
		parts, donorParts := strings.Split(token, "."), strings.Split(donor, ".")
		return parts[0] + "." + donorParts[1] + "." + parts[2]
	}

	tests := []struct {
		name    string
		token   string
		now     time.Time
		wantErr bool
	}{
		{name: "token válido", token: token, now: issuedAt.Add(time.Hour)},
		{name: "expirado", token: token, now: issuedAt.Add(24*time.Hour + time.Second), wantErr: true},
		{name: "firmado con otro secreto", token: foreignToken, now: issuedAt, wantErr: true},
		{name: "payload alterado", token: tamper(token, sign(jwt.SigningMethodHS256, []byte(strings.Repeat("z", MinJWTSecretLen)), withClaims(func(c *jwt.RegisteredClaims) { c.Subject = domain.NewID().String() }))), now: issuedAt, wantErr: true},
		{name: "algoritmo none", token: sign(jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, validClaims()), now: issuedAt, wantErr: true},
		{name: "HS512 no permitido", token: sign(jwt.SigningMethodHS512, testSecret, validClaims()), now: issuedAt, wantErr: true},
		{name: "otro emisor", token: sign(jwt.SigningMethodHS256, testSecret, withClaims(func(c *jwt.RegisteredClaims) { c.Issuer = "otro" })), now: issuedAt, wantErr: true},
		{name: "sin expiración", token: sign(jwt.SigningMethodHS256, testSecret, withClaims(func(c *jwt.RegisteredClaims) { c.ExpiresAt = nil })), now: issuedAt, wantErr: true},
		{name: "sujeto no UUID", token: sign(jwt.SigningMethodHS256, testSecret, withClaims(func(c *jwt.RegisteredClaims) { c.Subject = "admin" })), now: issuedAt, wantErr: true},
		{name: "emitido en el futuro", token: token, now: issuedAt.Add(-time.Hour), wantErr: true},
		{name: "basura", token: "no.es.jwt", now: issuedAt, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clock = tt.now
			got, err := j.Verify(tt.token)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidToken) {
					t.Fatalf("Verify() error = %v, se esperaba ErrInvalidToken", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Verify() error inesperado: %v", err)
			}
			if got != userID {
				t.Errorf("Verify() = %v, se esperaba %v", got, userID)
			}
		})
	}
}
