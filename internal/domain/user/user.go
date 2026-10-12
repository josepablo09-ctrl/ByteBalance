// Package user modela a los usuarios de ByteBalance y sus preferencias.
package user

import (
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"bytebalance/internal/domain"
)

// Límites alineados con las columnas de la tabla users.
const (
	maxEmailLen    = 255
	maxFullNameLen = 150
	maxTimezoneLen = 64
	defaultTZ      = "UTC"
)

// User es una persona registrada en la aplicación.
type User struct {
	ID              domain.ID
	Email           string
	PasswordHash    string
	FullName        string
	DefaultCurrency domain.Currency
	Timezone        string
	// TelegramChatID es nil mientras el usuario no haya vinculado su cuenta de Telegram.
	TelegramChatID *int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}

// Params agrupa los datos necesarios para crear un User.
type Params struct {
	Email string
	// PasswordHash debe llegar ya hasheado; el dominio nunca maneja contraseñas en claro.
	PasswordHash    string
	FullName        string
	DefaultCurrency domain.Currency
	// Timezone es un nombre IANA (p. ej. "America/Costa_Rica"); vacío equivale a UTC.
	Timezone string
}

// New normaliza los datos de p y devuelve un User válido sin persistir.
func New(p Params) (*User, error) {
	u := &User{
		Email:           strings.ToLower(strings.TrimSpace(p.Email)),
		PasswordHash:    p.PasswordHash,
		FullName:        strings.TrimSpace(p.FullName),
		DefaultCurrency: p.DefaultCurrency,
		Timezone:        strings.TrimSpace(p.Timezone),
	}
	if u.Timezone == "" {
		u.Timezone = defaultTZ
	}
	if err := u.Validate(); err != nil {
		return nil, err
	}
	return u, nil
}

// Validate comprueba las invariantes del usuario.
func (u *User) Validate() error {
	if u.Email == "" || len(u.Email) > maxEmailLen {
		return fmt.Errorf("%w: el email es obligatorio y admite hasta %d caracteres", domain.ErrInvalidInput, maxEmailLen)
	}
	if addr, err := mail.ParseAddress(u.Email); err != nil || addr.Address != u.Email {
		return fmt.Errorf("%w: email con formato inválido", domain.ErrInvalidInput)
	}
	if u.PasswordHash == "" {
		return fmt.Errorf("%w: falta el hash de la contraseña", domain.ErrInvalidInput)
	}
	if u.FullName == "" || utf8.RuneCountInString(u.FullName) > maxFullNameLen {
		return fmt.Errorf("%w: el nombre es obligatorio y admite hasta %d caracteres", domain.ErrInvalidInput, maxFullNameLen)
	}
	if !u.DefaultCurrency.Valid() {
		return fmt.Errorf("%w: moneda por defecto %q inválida", domain.ErrInvalidInput, u.DefaultCurrency)
	}
	// time.LoadLocation("") devuelve UTC sin error, por eso el vacío se rechaza aparte.
	if u.Timezone == "" || len(u.Timezone) > maxTimezoneLen {
		return fmt.Errorf("%w: la zona horaria es obligatoria y admite hasta %d caracteres", domain.ErrInvalidInput, maxTimezoneLen)
	}
	if _, err := time.LoadLocation(u.Timezone); err != nil {
		return fmt.Errorf("%w: zona horaria %q desconocida", domain.ErrInvalidInput, u.Timezone)
	}
	return nil
}

// Location devuelve la zona horaria del usuario, usada para calcular sus días y reportes.
func (u *User) Location() (*time.Location, error) {
	loc, err := time.LoadLocation(u.Timezone)
	if err != nil {
		return nil, fmt.Errorf("cargar zona horaria del usuario: %w", err)
	}
	return loc, nil
}

// LinkTelegram asocia el chat de Telegram donde el usuario recibirá notificaciones.
func (u *User) LinkTelegram(chatID int64) error {
	if chatID == 0 {
		return fmt.Errorf("%w: chat de Telegram inválido", domain.ErrInvalidInput)
	}
	u.TelegramChatID = &chatID
	return nil
}

// UnlinkTelegram elimina la vinculación con Telegram.
func (u *User) UnlinkTelegram() {
	u.TelegramChatID = nil
}

// HasTelegram indica si el usuario puede recibir notificaciones por Telegram.
func (u *User) HasTelegram() bool {
	return u.TelegramChatID != nil
}
