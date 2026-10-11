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
