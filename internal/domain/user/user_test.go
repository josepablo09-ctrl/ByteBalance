package user

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

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
		{name: "zona horaria demasiado larga", modify: func(p *Params) { p.Timezone = strings.Repeat("a", 65) }, wantErr: true},
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

func TestValidateRejectsEmptyTimezone(t *testing.T) {
	u, err := New(validParams())
	if err != nil {
		t.Fatalf("New() error inesperado: %v", err)
	}
	u.Timezone = ""
	if err := u.Validate(); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("Validate() con zona horaria vacía error = %v, se esperaba ErrInvalidInput", err)
	}
}

func TestNewLinkCode(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	id := domain.NewID()

	code, lc, err := NewLinkCode(id, now)
	if err != nil {
		t.Fatalf("NewLinkCode() error inesperado: %v", err)
	}
	if len(code) != 32 || strings.Trim(code, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-") != "" {
		t.Errorf("código %q: debe tener 32 caracteres válidos para el parámetro start de Telegram", code)
	}
	if lc.UserID != id || !lc.ExpiresAt.Equal(now.Add(LinkCodeTTL)) {
		t.Errorf("LinkCode = %+v", lc)
	}
	if !bytes.Equal(lc.Hash, HashLinkCode(code)) || bytes.Contains(lc.Hash, []byte(code)) {
		t.Error("solo debe persistirse el hash del código")
	}

	other, _, err := NewLinkCode(id, now)
	if err != nil || other == code {
		t.Errorf("dos códigos generados no deben coincidir (%q, %v)", other, err)
	}
	if _, _, err := NewLinkCode(domain.ID{}, now); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("sin usuario: error = %v, se esperaba ErrInvalidInput", err)
	}
}

func TestValidateSavingsGoal(t *testing.T) {
	cents := func(c int64) *int64 { return &c }
	tests := []struct {
		name    string
		goal    *int64
		wantErr bool
	}{
		{name: "sin meta", goal: nil},
		{name: "meta válida", goal: cents(100_000_00)},
		{name: "meta máxima", goal: cents(99_999_999_999_999)},
		{name: "meta cero", goal: cents(0), wantErr: true},
		{name: "meta negativa", goal: cents(-1), wantErr: true},
		{name: "meta que no cabe en NUMERIC(14,2)", goal: cents(100_000_000_000_000), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := New(validParams())
			if err != nil {
				t.Fatalf("New() error inesperado: %v", err)
			}
			u.MonthlySavingsGoalCents = tt.goal
			err = u.Validate()
			if tt.wantErr != errors.Is(err, domain.ErrInvalidInput) || (!tt.wantErr && err != nil) {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
