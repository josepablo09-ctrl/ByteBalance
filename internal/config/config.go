// Package config carga y valida la configuración de ByteBalance desde
// variables de entorno. Es el único lugar de la aplicación que las lee.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
)

// Entornos de ejecución admitidos en APP_ENV.
const (
	EnvDevelopment = "development"
	EnvStaging     = "staging"
	EnvProduction  = "production"
)

const (
	defaultEnv  = EnvDevelopment
	defaultPort = 8080
	dotEnvFile  = ".env"
	// minJWTSecretLen coincide con security.MinJWTSecretLen (256 bits para HS256).
	minJWTSecretLen = 32
	// maxTrustedProxyHops acota TRUSTED_PROXY_HOPS a un valor razonable.
	maxTrustedProxyHops = 10
)

// sslmodes que cifran la conexión; obligatorios fuera de desarrollo.
var secureSSLModes = []string{"require", "verify-ca", "verify-full"}

// Config es la configuración validada de la aplicación.
type Config struct {
	// Env es el entorno de ejecución (APP_ENV).
	Env string
	// Port es el puerto HTTP del backend (APP_PORT).
	Port int
	// DatabaseURL es la cadena de conexión a PostgreSQL (DATABASE_URL). Contiene
	// credenciales: nunca debe registrarse en logs.
	DatabaseURL string
	// JWTSecret es la clave HS256 con la que se firman los tokens de acceso
	// (JWT_SECRET). Es un secreto: nunca debe registrarse en logs.
	JWTSecret string
	// TrustedProxyHops es el número de proxies de confianza delante de la API
	// (TRUSTED_PROXY_HOPS), p. ej. 1 detrás de un ALB de AWS. Determina qué
	// entrada de X-Forwarded-For es la IP real del cliente. Por defecto 0: la
	// cabecera se ignora.
	TrustedProxyHops int
	// CORSAllowedOrigins son los orígenes del frontend que pueden llamar a la
	// API desde el navegador (CORS_ALLOWED_ORIGINS, separados por comas). Vacío
	// desactiva CORS: solo se admiten peticiones del mismo origen.
	CORSAllowedOrigins []string
}

// IsDevelopment indica si la aplicación corre en el entorno local de desarrollo.
func (c Config) IsDevelopment() bool {
	return c.Env == EnvDevelopment
}

// Addr devuelve la dirección de escucha del servidor HTTP.
func (c Config) Addr() string {
	return ":" + strconv.Itoa(c.Port)
}

// Load lee la configuración desde las variables de entorno y la valida. En
// desarrollo (APP_ENV vacío o "development") también lee el archivo .env del
// directorio de trabajo, si existe; las variables del entorno tienen prioridad
// sobre las del archivo. Devuelve todos los problemas encontrados a la vez.
func Load() (Config, error) {
	lookup := os.LookupEnv
	if env := os.Getenv("APP_ENV"); env == "" || env == EnvDevelopment {
		fileVars, err := readDotEnv(dotEnvFile)
		if err != nil {
			return Config{}, err
		}
		lookup = withFallback(os.LookupEnv, fileVars)
	}
	return parse(lookup)
}

// withFallback consulta primero lookup y, si la variable no está definida, fallback.
func withFallback(lookup func(string) (string, bool), fallback map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		if v, ok := lookup(key); ok {
			return v, true
		}
		v, ok := fallback[key]
		return v, ok
	}
}

// parse construye y valida un Config a partir de lookup.
func parse(lookup func(string) (string, bool)) (Config, error) {
	get := func(key string) string {
		if v, ok := lookup(key); ok {
			return v
		}
		return "" // una variable no definida equivale a vacía
	}
	var errs []error
	cfg := Config{
		Env:         get("APP_ENV"),
		Port:        defaultPort,
		DatabaseURL: get("DATABASE_URL"),
		JWTSecret:   get("JWT_SECRET"),
	}

	if cfg.Env == "" {
		cfg.Env = defaultEnv
	}
	if !slices.Contains([]string{EnvDevelopment, EnvStaging, EnvProduction}, cfg.Env) {
		errs = append(errs, fmt.Errorf("APP_ENV %q inválido: usar development, staging o production", cfg.Env))
	}

	if raw := get("APP_PORT"); raw != "" {
		port, err := strconv.Atoi(raw)
		if err != nil || port < 1 || port > 65535 {
			errs = append(errs, fmt.Errorf("APP_PORT %q inválido: debe ser un número entre 1 y 65535", raw))
		}
		cfg.Port = port
	}

	if raw := get("TRUSTED_PROXY_HOPS"); raw != "" {
		hops, err := strconv.Atoi(raw)
		if err != nil || hops < 0 || hops > maxTrustedProxyHops {
			errs = append(errs, fmt.Errorf("TRUSTED_PROXY_HOPS %q inválido: debe ser un número entre 0 y %d", raw, maxTrustedProxyHops))
		}
		cfg.TrustedProxyHops = hops
	}

	if err := validateDatabaseURL(cfg.DatabaseURL, cfg.Env); err != nil {
		errs = append(errs, err)
	}

	origins, err := parseOrigins(get("CORS_ALLOWED_ORIGINS"), cfg.Env)
	if err != nil {
		errs = append(errs, err)
	}
	cfg.CORSAllowedOrigins = origins

	// No se incluye el valor en el mensaje: es un secreto.
	if len(cfg.JWTSecret) < minJWTSecretLen {
		errs = append(errs, fmt.Errorf("JWT_SECRET es obligatoria y debe tener al menos %d bytes (generar con: openssl rand -base64 48)", minJWTSecretLen))
	}

	if err := errors.Join(errs...); err != nil {
		return Config{}, fmt.Errorf("configuración inválida:\n%w", err)
	}
	return cfg, nil
}

// parseOrigins interpreta una lista de orígenes separados por comas. Cada uno
// debe ser exactamente esquema://host[:puerto], sin ruta ni comodines, porque
// el navegador envía la cabecera Origin en esa forma y se compara literalmente.
// Fuera de desarrollo solo se admite https.
func parseOrigins(raw, env string) ([]string, error) {
	var origins []string
	for _, item := range strings.Split(raw, ",") {
		origin := strings.ToLower(strings.TrimSpace(item))
		if origin == "" {
			continue
		}
		u, err := url.Parse(origin)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" ||
			u.User != nil || origin != u.Scheme+"://"+u.Host {
			return nil, fmt.Errorf("CORS_ALLOWED_ORIGINS: %q no es un origen válido (usar esquema://host[:puerto], sin ruta ni \"*\")", item)
		}
		if env != EnvDevelopment && u.Scheme != "https" {
			return nil, fmt.Errorf("CORS_ALLOWED_ORIGINS: %q debe usar https en %s", item, env)
		}
		if !slices.Contains(origins, origin) {
			origins = append(origins, origin)
		}
	}
	return origins, nil
}

// validateDatabaseURL comprueba DATABASE_URL sin incluir su valor en los
// errores, porque contiene la contraseña.
func validateDatabaseURL(raw, env string) error {
	if raw == "" {
		return errors.New("DATABASE_URL es obligatoria")
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Host == "" {
		return errors.New("DATABASE_URL debe tener la forma postgres://usuario:clave@host:puerto/base")
	}
	if env != EnvDevelopment && !slices.Contains(secureSSLModes, u.Query().Get("sslmode")) {
		return fmt.Errorf("DATABASE_URL debe usar sslmode=require (o verify-ca/verify-full) en %s", env)
	}
	return nil
}
