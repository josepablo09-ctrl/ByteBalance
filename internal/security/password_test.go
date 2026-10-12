package security

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestNewBcryptHasher(t *testing.T) {
	tests := []struct {
		name    string
		cost    int
		wantErr bool
	}{
		{name: "costo por defecto", cost: DefaultBcryptCost},
		{name: "costo mínimo", cost: bcrypt.MinCost},
		{name: "costo demasiado bajo", cost: bcrypt.MinCost - 1, wantErr: true},
		{name: "costo demasiado alto", cost: bcrypt.MaxCost + 1, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewBcryptHasher(tt.cost)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewBcryptHasher(%d) error = %v, wantErr %v", tt.cost, err, tt.wantErr)
			}
		})
	}
}

func TestBcryptHasher(t *testing.T) {
	// Costo mínimo para que el test sea rápido; el algoritmo es el mismo.
	h, err := NewBcryptHasher(bcrypt.MinCost)
	if err != nil {
		t.Fatalf("NewBcryptHasher: %v", err)
	}
	hash, err := h.Hash("contraseña-segura")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if hash == "contraseña-segura" || !strings.HasPrefix(hash, "$2a$") {
		t.Fatalf("Hash() = %q, se esperaba un hash bcrypt", hash)
	}
	other, err := h.Hash("contraseña-segura")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if other == hash {
		t.Error("dos hashes de la misma contraseña deben diferir (sal aleatoria)")
	}

	tests := []struct {
		name     string
		hash     string
		password string
		wantOK   bool
		wantErr  bool
	}{
		{name: "contraseña correcta", hash: hash, password: "contraseña-segura", wantOK: true},
		{name: "contraseña incorrecta", hash: hash, password: "otra-contraseña"},
		{name: "distingue mayúsculas", hash: hash, password: "Contraseña-segura"},
		{name: "hash inválido", hash: "no-es-bcrypt", password: "contraseña-segura", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := h.Verify(tt.hash, tt.password)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Verify() error = %v, wantErr %v", err, tt.wantErr)
			}
			if ok != tt.wantOK {
				t.Errorf("Verify() = %v, se esperaba %v", ok, tt.wantOK)
			}
		})
	}

	if _, err := h.Hash(strings.Repeat("a", 73)); err == nil {
		t.Error("Hash() debe rechazar contraseñas de más de 72 bytes")
	}
}
