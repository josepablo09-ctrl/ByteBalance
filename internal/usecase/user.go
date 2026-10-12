package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/user"
)

// Política de contraseñas (NIST SP 800-63B): solo longitud, sin reglas de
// composición. El máximo se mide en bytes porque los algoritmos de hash como
// bcrypt no admiten más de 72 bytes; un carácter no ASCII ocupa varios.
const (
	minPasswordLen   = 8
	maxPasswordBytes = 72
)

const defaultCurrency domain.Currency = "USD"

// ErrEmailTaken indica que ya existe un usuario activo con ese email.
var ErrEmailTaken = fmt.Errorf("%w: el email ya está registrado", domain.ErrConflict)

// PasswordHasher genera el hash de una contraseña. La implementación debe usar
// un algoritmo lento y con sal (argon2id, bcrypt o PBKDF2).
type PasswordHasher interface {
	Hash(password string) (string, error)
}

// UserService implementa los casos de uso de registro y preferencias de usuario.
type UserService struct {
	users  user.Repository
	hasher PasswordHasher
}

// NewUserService crea un UserService.
func NewUserService(users user.Repository, hasher PasswordHasher) *UserService {
	return &UserService{users: users, hasher: hasher}
}

// RegisterInput son los datos de registro de un usuario.
type RegisterInput struct {
	Email string
	// Password llega en claro; solo se conserva su hash.
	Password string
	FullName string
	// DefaultCurrency vacía equivale a USD.
	DefaultCurrency string
	// Timezone vacía equivale a UTC.
	Timezone string
}

// Register crea un usuario nuevo. Devuelve ErrEmailTaken si el email ya está en uso.
func (s *UserService) Register(ctx context.Context, in RegisterInput) (*user.User, error) {
	if utf8.RuneCountInString(in.Password) < minPasswordLen || len(in.Password) > maxPasswordBytes {
		return nil, fmt.Errorf("registrar usuario: %w: la contraseña debe tener al menos %d caracteres y no superar %d bytes",
			domain.ErrInvalidInput, minPasswordLen, maxPasswordBytes)
	}
	currency := defaultCurrency
	if strings.TrimSpace(in.DefaultCurrency) != "" {
		c, err := domain.ParseCurrency(in.DefaultCurrency)
		if err != nil {
			return nil, fmt.Errorf("registrar usuario: %w", err)
		}
		currency = c
	}

	hash, err := s.hasher.Hash(in.Password)
	if err != nil {
		return nil, fmt.Errorf("registrar usuario: hashear contraseña: %w", err)
	}
	u, err := user.New(user.Params{
		Email:           in.Email,
		PasswordHash:    hash,
		FullName:        in.FullName,
		DefaultCurrency: currency,
		Timezone:        in.Timezone,
	})
	if err != nil {
		return nil, fmt.Errorf("registrar usuario: %w", err)
	}

	if err := s.users.Create(ctx, u); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("registrar usuario: %w", err)
	}
	return u, nil
}

// PreferencesInput son las preferencias a modificar. Un campo nil se deja igual.
type PreferencesInput struct {
	FullName        *string
	DefaultCurrency *string
	Timezone        *string
}

// UpdatePreferences modifica el nombre, la moneda por defecto o la zona horaria
// del usuario. Si algún valor es inválido no se guarda ningún cambio.
func (s *UserService) UpdatePreferences(ctx context.Context, userID domain.ID, in PreferencesInput) (*user.User, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("actualizar preferencias: %w", err)
	}
	if in.FullName != nil {
		u.FullName = strings.TrimSpace(*in.FullName)
	}
	if in.DefaultCurrency != nil {
		c, err := domain.ParseCurrency(*in.DefaultCurrency)
		if err != nil {
			return nil, fmt.Errorf("actualizar preferencias: %w", err)
		}
		u.DefaultCurrency = c
	}
	if in.Timezone != nil {
		u.Timezone = strings.TrimSpace(*in.Timezone)
	}
	if err := u.Validate(); err != nil {
		return nil, fmt.Errorf("actualizar preferencias: %w", err)
	}

	if err := s.users.Update(ctx, u); err != nil {
		return nil, fmt.Errorf("actualizar preferencias: %w", err)
	}
	return u, nil
}
