// Package config carrega e valida a configuração da aplicação a partir de variáveis de ambiente.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Env              string
	HTTPPort         string
	DatabaseURL      string
	SessionSecret    string
	CookieSecure     bool
	CORSOrigins      []string
	HTTPReadTimeout  time.Duration
	HTTPWriteTimeout time.Duration
	HTTPIdleTimeout  time.Duration
	DBMaxConns       int32
	DBConnTimeout    time.Duration
}

// Load lê as variáveis de ambiente obrigatórias e opcionais, aplicando valores
// padrão seguros apenas para configurações não sensíveis. Retorna erro
// descritivo quando uma variável obrigatória está ausente ou inválida.
func Load() (*Config, error) {
	cfg := &Config{
		Env:              getEnv("APP_ENV", "development"),
		HTTPPort:         getEnv("HTTP_PORT", "8080"),
		CookieSecure:     getEnvBool("COOKIE_SECURE", false),
		HTTPReadTimeout:  getEnvDuration("HTTP_READ_TIMEOUT", 10*time.Second),
		HTTPWriteTimeout: getEnvDuration("HTTP_WRITE_TIMEOUT", 15*time.Second),
		HTTPIdleTimeout:  getEnvDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
		DBConnTimeout:    getEnvDuration("DB_CONN_TIMEOUT", 5*time.Second),
	}

	dbURL, ok := os.LookupEnv("DATABASE_URL")
	if !ok || dbURL == "" {
		return nil, fmt.Errorf("variável de ambiente obrigatória ausente: DATABASE_URL")
	}
	cfg.DatabaseURL = dbURL

	secret, ok := os.LookupEnv("SESSION_SECRET")
	if !ok || len(secret) < 32 {
		return nil, fmt.Errorf("variável de ambiente obrigatória ausente ou muito curta (mínimo 32 caracteres): SESSION_SECRET")
	}
	cfg.SessionSecret = secret

	origins := getEnv("CORS_ALLOWED_ORIGINS", "")
	cfg.CORSOrigins = splitAndTrim(origins)

	maxConns, err := strconv.Atoi(getEnv("DB_MAX_CONNS", "10"))
	if err != nil || maxConns <= 0 {
		return nil, fmt.Errorf("variável de ambiente inválida: DB_MAX_CONNS deve ser um inteiro positivo")
	}
	cfg.DBMaxConns = int32(maxConns)

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func splitAndTrim(s string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			part := trimSpace(s[start:i])
			if part != "" {
				out = append(out, part)
			}
			start = i + 1
		}
	}
	return out
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
