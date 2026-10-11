package domain

import (
	"errors"
	"testing"
)

func TestParseCurrency(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Currency
		wantErr bool
	}{
		{name: "válida", input: "USD", want: "USD"},
		{name: "normaliza minúsculas y espacios", input: " crc ", want: "CRC"},
		{name: "vacía", input: "", wantErr: true},
		{name: "muy corta", input: "US", wantErr: true},
		{name: "muy larga", input: "USDT", wantErr: true},
		{name: "con dígitos", input: "U5D", wantErr: true},
		{name: "no ASCII", input: "ÑAB", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCurrency(tt.input)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidInput) {
					t.Fatalf("ParseCurrency(%q) error = %v, se esperaba ErrInvalidInput", tt.input, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseCurrency(%q) error inesperado: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ParseCurrency(%q) = %q, se esperaba %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "válido", input: "0199d3a4-5b6c-7d8e-9f01-23456789abcd", want: "0199d3a4-5b6c-7d8e-9f01-23456789abcd"},
		{name: "mayúsculas se normalizan", input: "0199D3A4-5B6C-7D8E-9F01-23456789ABCD", want: "0199d3a4-5b6c-7d8e-9f01-23456789abcd"},
		{name: "vacío", input: "", wantErr: true},
		{name: "formato inválido", input: "no-es-un-uuid", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseID(tt.input)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidInput) {
					t.Fatalf("ParseID(%q) error = %v, se esperaba ErrInvalidInput", tt.input, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseID(%q) error inesperado: %v", tt.input, err)
			}
			if got.String() != tt.want {
				t.Errorf("ParseID(%q) = %q, se esperaba %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIDIsZero(t *testing.T) {
	if !(ID{}).IsZero() {
		t.Error("el ID cero debe reportar IsZero")
	}
	if NewID().IsZero() {
		t.Error("NewID no debe devolver el ID cero")
	}
}

func TestIDTextRoundTrip(t *testing.T) {
	id := NewID()
	text, err := id.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText() error inesperado: %v", err)
	}
	var got ID
	if err := got.UnmarshalText(text); err != nil {
		t.Fatalf("UnmarshalText() error inesperado: %v", err)
	}
	if got != id {
		t.Errorf("round trip = %v, se esperaba %v", got, id)
	}
	if err := got.UnmarshalText([]byte("basura")); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("UnmarshalText(basura) error = %v, se esperaba ErrInvalidInput", err)
	}
}
