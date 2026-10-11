package finance

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"bytebalance/internal/domain"
)

const maxCategoryNameLen = 80

// Category agrupa transacciones de un usuario bajo un nombre y un tipo.
type Category struct {
	ID        domain.ID
	UserID    domain.ID
	Name      string
	Type      EntryType
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// NewCategory devuelve una Category válida sin persistir.
func NewCategory(userID domain.ID, name string, t EntryType) (*Category, error) {
	c := &Category{UserID: userID, Name: strings.TrimSpace(name), Type: t}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

// Validate comprueba las invariantes de la categoría.
func (c *Category) Validate() error {
	if c.UserID.IsZero() {
		return fmt.Errorf("%w: la categoría debe pertenecer a un usuario", domain.ErrInvalidInput)
	}
	if c.Name == "" || utf8.RuneCountInString(c.Name) > maxCategoryNameLen {
		return fmt.Errorf("%w: el nombre de la categoría es obligatorio y admite hasta %d caracteres", domain.ErrInvalidInput, maxCategoryNameLen)
	}
	if !c.Type.Valid() {
		return fmt.Errorf("%w: tipo de categoría %q inválido", domain.ErrInvalidInput, c.Type)
	}
	return nil
}
