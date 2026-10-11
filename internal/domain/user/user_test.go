package user

import (
	"errors"
	"strings"
	"testing"

	"bytebalance/internal/domain"
)

func validParams() Params {
	return Params{
		Email:           "Ana@Example.com ",
		PasswordHash:    "hash",
		FullName:        " Ana Pérez ",
		DefaultCurrency: "CRC",
		Timezone:        "America/Costa_Rica",
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(p *Params)
		wantErr bool
	}{
		{name: "válido", modify: func(*Params) {}},
		{name: "zona horaria vacía usa UTC", modify: func(p *Params) { p.Timezone = "" }},
		{name: "email vacío", modify: func(p *Params) { p.Email = "  " }, wantErr: true},
		{name: "email sin arroba", modify: func(p *Params) { p.Email = "ana.example.com" }, wantErr: true},
		{name: "email con nombre visible", modify: func(p *Params) { p.Email = "Ana <ana@example.com>" }, wantErr: true},
		{name: "email demasiado largo", modify: func(p *Params) { p.Email = strings.Repeat("a", 250) + "@x.com" }, wantErr: true},
		{name: "sin hash", modify: func(p *Params) { p.PasswordHash = "" }, wantErr: true},
		{name: "nombre vacío", modify: func(p *Params) { p.FullName = " " }, wantErr: true},
		{name: "nombre demasiado largo", modify: func(p *Params) { p.FullName = strings.Repeat("ñ", 151) }, wantErr: true},
		{name: "moneda inválida", modify: func(p *Params) { p.DefaultCurrency = "usd" }, wantErr: true},
		{name: "zona horaria desconocida", modify: func(p *Params) { p.Timezone = "Marte/Olympus" }, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := validParams()
			tt.modify(&p)
			u, err := New(p)
			if tt.wantErr {
				if !errors.Is(err, domain.ErrInvalidInput) {
					t.Fatalf("New() error = %v, se esperaba ErrInvalidInput", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("New() error inesperado: %v", err)
			}
			if u.Email != "ana@example.com" {
				t.Errorf("Email = %q, se esperaba normalizado", u.Email)
			}
			if u.FullName != "Ana Pérez" {
				t.Errorf("FullName = %q, se esperaba sin espacios", u.FullName)
			}
			if p.Timezone == "" && u.Timezone != "UTC" {
				t.Errorf("Timezone = %q, se esperaba UTC", u.Timezone)
			}
		})
	}
}

func TestLinkTelegram(t *testing.T) {
	u, err := New(validParams())
	if err != nil {
		t.Fatalf("New() error inesperado: %v", err)
	}
	if err := u.LinkTelegram(0); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("LinkTelegram(0) error = %v, se esperaba ErrInvalidInput", err)
	}
	if err := u.LinkTelegram(12345); err != nil {
		t.Fatalf("LinkTelegram() error inesperado: %v", err)
	}
	if !u.HasTelegram() || *u.TelegramChatID != 12345 {
		t.Fatalf("TelegramChatID no quedó vinculado")
	}
	u.UnlinkTelegram()
	if u.HasTelegram() {
		t.Fatalf("UnlinkTelegram() no eliminó la vinculación")
	}
}
