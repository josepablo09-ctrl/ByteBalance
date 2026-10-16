package user

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"bytebalance/internal/domain"
)

// LinkCodeTTL es la vigencia de un código para vincular la cuenta con Telegram.
const LinkCodeTTL = 10 * time.Minute

// linkCodeBytes es la entropía del código: 192 bits, imposible de adivinar
// dentro de su vigencia.
const linkCodeBytes = 24

// LinkCode es un código de un solo uso para vincular la cuenta con un chat de
// Telegram. Solo se persiste el hash: el código en claro se entrega una vez al
// usuario dentro del enlace al bot.
type LinkCode struct {
	UserID    domain.ID
	Hash      []byte
	ExpiresAt time.Time
}

// NewLinkCode genera un código aleatorio para userID, vigente hasta
// now+LinkCodeTTL. Devuelve el código en claro y su versión persistible. El
// código solo usa [A-Za-z0-9_-], lo que admite el parámetro start de Telegram.
func NewLinkCode(userID domain.ID, now time.Time) (string, *LinkCode, error) {
	if userID.IsZero() {
		return "", nil, fmt.Errorf("%w: el código debe pertenecer a un usuario", domain.ErrInvalidInput)
	}
	buf := make([]byte, linkCodeBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", nil, fmt.Errorf("generar código de vinculación: %w", err)
	}
	code := base64.RawURLEncoding.EncodeToString(buf)
	return code, &LinkCode{UserID: userID, Hash: HashLinkCode(code), ExpiresAt: now.Add(LinkCodeTTL)}, nil
}

// HashLinkCode devuelve el SHA-256 del código. Un hash rápido basta (a
// diferencia de una contraseña) porque el código es aleatorio y vence en minutos.
func HashLinkCode(code string) []byte {
	sum := sha256.Sum256([]byte(code))
	return sum[:]
}
