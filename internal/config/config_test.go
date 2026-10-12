package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const localDB = "postgres://user:password@localhost:5432/bytebalance?sslmode=disable"

func lookupFrom(vars map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		v, ok := vars[key]
		return v, ok
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		vars    map[string]string
		want    Config
		wantErr []string // fragmentos que debe contener el error
	}{
		{
			name: "valores por defecto",
			vars: map[string]string{"DATABASE_URL": localDB},
			want: Config{Env: EnvDevelopment, Port: 8080, DatabaseURL: localDB},
		},
		{
			name: "producción con TLS",
			vars: map[string]string{
				"APP_ENV":      "production",
				"APP_PORT":     "9090",
				"DATABASE_URL": "postgres://app:secreto@db.internal:5432/bytebalance?sslmode=verify-full",
			},
			want: Config{Env: EnvProduction, Port: 9090, DatabaseURL: "postgres://app:secreto@db.internal:5432/bytebalance?sslmode=verify-full"},
		},
		{
			name:    "falta DATABASE_URL",
			vars:    map[string]string{},
			wantErr: []string{"DATABASE_URL es obligatoria"},
		},
		{
			name:    "DATABASE_URL con otro esquema",
			vars:    map[string]string{"DATABASE_URL": "mysql://user:password@localhost/db"},
			wantErr: []string{"DATABASE_URL debe tener la forma"},
		},
		{
			name:    "staging sin TLS",
			vars:    map[string]string{"APP_ENV": "staging", "DATABASE_URL": localDB},
			wantErr: []string{"sslmode=require"},
		},
		{
			name:    "reporta todos los errores a la vez",
			vars:    map[string]string{"APP_ENV": "qa", "APP_PORT": "70000"},
			wantErr: []string{"APP_ENV", "APP_PORT", "DATABASE_URL"},
		},
		{
			name:    "puerto no numérico",
			vars:    map[string]string{"APP_PORT": "http", "DATABASE_URL": localDB},
			wantErr: []string{"APP_PORT"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parse(lookupFrom(tt.vars))
			if len(tt.wantErr) > 0 {
				if err == nil {
					t.Fatalf("parse() = %+v, se esperaba error", got)
				}
				for _, fragment := range tt.wantErr {
					if !strings.Contains(err.Error(), fragment) {
						t.Errorf("error %q no menciona %q", err, fragment)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("parse() error inesperado: %v", err)
			}
			if got != tt.want {
				t.Errorf("parse() = %+v, se esperaba %+v", got, tt.want)
			}
		})
	}
}

func TestParseNeverLeaksDatabasePassword(t *testing.T) {
	_, err := parse(lookupFrom(map[string]string{
		"APP_ENV":      "production",
		"DATABASE_URL": "postgres://app:superSecreta@db.internal/bytebalance?sslmode=disable",
	}))
	if err == nil {
		t.Fatal("se esperaba error por falta de TLS")
	}
	if strings.Contains(err.Error(), "superSecreta") {
		t.Fatalf("el error expone la contraseña: %v", err)
	}
}

func TestParseDotEnv(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    map[string]string
		wantErr bool
	}{
		{
			name: "formato de .env.example",
			content: "# comentario\n\nAPP_ENV=development\nPOSTGRES_USER=user               # Solo para docker-compose\n" +
				"DATABASE_URL=postgres://u:p@localhost:5432/db?sslmode=disable\nGEMINI_API_KEY=\n",
			want: map[string]string{
				"APP_ENV":        "development",
				"POSTGRES_USER":  "user",
				"DATABASE_URL":   "postgres://u:p@localhost:5432/db?sslmode=disable",
				"GEMINI_API_KEY": "",
			},
		},
		{
			name:    "comillas y export",
			content: "export A=\"con # y espacios\"\nB='simple'\nC=valor#sin-espacio\n",
			want:    map[string]string{"A": "con # y espacios", "B": "simple", "C": "valor#sin-espacio"},
		},
		{name: "línea sin igual", content: "SOLO_CLAVE\n", wantErr: true},
		{name: "clave con espacios", content: "MI CLAVE=x\n", wantErr: true},
		{name: "comillas sin cerrar", content: "A=\"abierta\n", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDotEnv(bufio.NewScanner(strings.NewReader(tt.content)))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseDotEnv() = %v, se esperaba error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseDotEnv() error inesperado: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("parseDotEnv() = %v, se esperaba %v", got, tt.want)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("%s = %q, se esperaba %q", k, got[k], v)
				}
			}
		})
	}
}

func TestReadDotEnvMissingFile(t *testing.T) {
	vars, err := readDotEnv(filepath.Join(t.TempDir(), ".env"))
	if err != nil || len(vars) != 0 {
		t.Fatalf("readDotEnv() = %v, %v; se esperaba mapa vacío sin error", vars, err)
	}
}

func TestLoadEnvironmentOverridesDotEnv(t *testing.T) {
	dir := t.TempDir()
	content := "APP_PORT=7000\nDATABASE_URL=" + localDB + "\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(content), 0o600); err != nil {
		t.Fatalf("escribir .env: %v", err)
	}
	t.Chdir(dir)
	t.Setenv("APP_ENV", "")
	t.Setenv("APP_PORT", "9000")
	t.Setenv("DATABASE_URL", "")

	cfg, err := Load()
	// DATABASE_URL está definida (vacía) en el entorno y tiene prioridad sobre el archivo.
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL es obligatoria") {
		t.Fatalf("Load() = %+v, %v; se esperaba error por DATABASE_URL vacía", cfg, err)
	}

	if err := os.Unsetenv("DATABASE_URL"); err != nil {
		t.Fatalf("unsetenv: %v", err)
	}
	cfg, err = Load()
	if err != nil {
		t.Fatalf("Load() error inesperado: %v", err)
	}
	if cfg.Port != 9000 || cfg.DatabaseURL != localDB {
		t.Errorf("Load() = %+v; se esperaba APP_PORT del entorno y DATABASE_URL del archivo", cfg)
	}
}
