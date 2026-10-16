package main

import (
	"strings"
	"testing"

	"bytebalance/internal/config"
)

func TestCheckSafe(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.Config
		wantErr string // fragmento del error; vacío = permitido
	}{
		{name: "localhost", cfg: config.Config{Env: config.EnvDevelopment, DatabaseURL: "postgres://user:password@localhost:5432/bytebalance"}},
		{name: "127.0.0.1", cfg: config.Config{Env: config.EnvDevelopment, DatabaseURL: "postgres://u:p@127.0.0.1:5432/db"}},
		{name: "IPv6 local", cfg: config.Config{Env: config.EnvDevelopment, DatabaseURL: "postgres://u:p@[::1]:5432/db"}},
		{
			name:    "producción",
			cfg:     config.Config{Env: config.EnvProduction, DatabaseURL: "postgres://u:p@localhost/db"},
			wantErr: "APP_ENV=development",
		},
		{
			name:    "staging",
			cfg:     config.Config{Env: config.EnvStaging, DatabaseURL: "postgres://u:p@localhost/db"},
			wantErr: "APP_ENV=development",
		},
		{
			name:    "base remota en desarrollo",
			cfg:     config.Config{Env: config.EnvDevelopment, DatabaseURL: "postgres://u:secreta@db.bytebalance.internal:5432/db"},
			wantErr: "db.bytebalance.internal",
		},
		{
			// Un host que solo empieza por "localhost" no es local.
			name:    "host engañoso",
			cfg:     config.Config{Env: config.EnvDevelopment, DatabaseURL: "postgres://u:p@localhost.evil.com/db"},
			wantErr: "localhost.evil.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkSafe(tt.cfg)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("checkSafe() error inesperado: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("checkSafe() error = %v, se esperaba que mencionara %q", err, tt.wantErr)
			}
			if strings.Contains(err.Error(), "secreta") {
				t.Errorf("el error expone la contraseña: %v", err)
			}
		})
	}
}

func TestE2EEmailPattern(t *testing.T) {
	// El patrón debe limitarse al dominio reservado de las pruebas.
	if !strings.HasPrefix(e2eEmailPattern, "e2e+") || !strings.HasSuffix(e2eEmailPattern, "@bytebalance.test") {
		t.Fatalf("patrón inesperado: %q", e2eEmailPattern)
	}
	if strings.Count(e2eEmailPattern, "%") != 1 || strings.Contains(e2eEmailPattern, "_") {
		t.Fatalf("el patrón no debe tener comodines adicionales: %q", e2eEmailPattern)
	}
}
