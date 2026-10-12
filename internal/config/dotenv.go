package config

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// readDotEnv lee un archivo .env y devuelve sus variables. Si el archivo no
// existe devuelve un mapa vacío. No modifica el entorno del proceso.
func readDotEnv(path string) (vars map[string]string, err error) {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("abrir %s: %w", path, err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			vars, err = nil, fmt.Errorf("cerrar %s: %w", path, cerr)
		}
	}()

	vars, err = parseDotEnv(bufio.NewScanner(f))
	if err != nil {
		return nil, fmt.Errorf("leer %s: %w", path, err)
	}
	return vars, nil
}

// parseDotEnv interpreta líneas KEY=VALUE. Admite líneas vacías, comentarios
// (# al inicio o tras un espacio en valores sin comillas), el prefijo "export"
// y valores entre comillas simples o dobles.
func parseDotEnv(sc *bufio.Scanner) (map[string]string, error) {
	vars := map[string]string{}
	for lineNum := 1; sc.Scan(); lineNum++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" || strings.ContainsAny(key, " \t") {
			// No se incluye el contenido de la línea: podría contener un secreto.
			return nil, fmt.Errorf("línea %d: se esperaba CLAVE=valor", lineNum)
		}
		value, err := parseDotEnvValue(strings.TrimSpace(value))
		if err != nil {
			return nil, fmt.Errorf("línea %d (%s): %w", lineNum, key, err)
		}
		vars[key] = value
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return vars, nil
}

// parseDotEnvValue quita las comillas o el comentario final de un valor.
func parseDotEnvValue(v string) (string, error) {
	if v != "" && (v[0] == '"' || v[0] == '\'') {
		end := strings.IndexByte(v[1:], v[0])
		if end < 0 {
			return "", errors.New("comillas sin cerrar")
		}
		return v[1 : end+1], nil
	}
	if i := strings.Index(v, " #"); i >= 0 {
		v = v[:i]
	}
	if i := strings.Index(v, "\t#"); i >= 0 {
		v = v[:i]
	}
	return strings.TrimSpace(v), nil
}
