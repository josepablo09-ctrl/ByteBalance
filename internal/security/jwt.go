package security

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"bytebalance/internal/domain"
)

// MinJWTSecretLen es la longitud mínima en bytes del secreto HS256 (256 bits).
const MinJWTSecretLen = 32

const jwtIssuer = "bytebalance"

// ErrInvalidToken indica que el token no es válido: mal formado, con firma
// incorrecta, expirado o emitido por otro sistema.
var ErrInvalidToken = errors.New("token inválido")

// JWT emite y verifica tokens de acceso JWT firmados con HS256. El sujeto
// (sub) del token es el ID del usuario.
type JWT struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// NewJWT crea un JWT. secret debe tener al menos MinJWTSecretLen bytes y ttl
// ser positivo. Si now es nil se usa time.Now.
func NewJWT(secret []byte, ttl time.Duration, now func() time.Time) (*JWT, error) {
	if len(secret) < MinJWTSecretLen {
		return nil, fmt.Errorf("el secreto JWT debe tener al menos %d bytes", MinJWTSecretLen)
	}
	if ttl <= 0 {
		return nil, errors.New("la duración del token debe ser positiva")
	}
	if now == nil {
		now = time.Now
	}
	return &JWT{secret: secret, ttl: ttl, now: now}, nil
}

// Issue emite un token de acceso para userID y devuelve también su expiración.
func (j *JWT) Issue(userID domain.ID) (string, time.Time, error) {
	now := j.now()
	expiresAt := now.Add(j.ttl)
	claims := jwt.RegisteredClaims{
		Issuer:    jwtIssuer,
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(j.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("firmar token: %w", err)
	}
	return token, expiresAt, nil
}

// Verify valida token y devuelve el ID del usuario. Solo acepta HS256, exige
// expiración y emisor, y devuelve un error que envuelve ErrInvalidToken si el
// token no es válido.
func (j *JWT) Verify(token string) (domain.ID, error) {
	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(token, claims,
		func(*jwt.Token) (any, error) { return j.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(jwtIssuer),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithTimeFunc(j.now),
	)
	if err != nil {
		return domain.ID{}, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}
	id, err := domain.ParseID(claims.Subject)
	if err != nil {
		return domain.ID{}, fmt.Errorf("%w: sujeto inválido", ErrInvalidToken)
	}
	return id, nil
}
