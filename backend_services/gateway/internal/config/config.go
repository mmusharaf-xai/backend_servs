package config

import (
	"os"
	"strings"
	"time"
)

type Config struct {
	Port               string
	Env                string
	DatabasesMemory    string // "LOCAL" = embedded in-memory databases, anything else = normal
	DatabaseURL        string
	RedisURL           string
	JWTSecret          string
	JWTAccessTTL       time.Duration
	RefreshTokenTTL    time.Duration
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	CORSAllowedOrigins []string
}

// IsLocalMemory returns true when databases should run embedded in-process
// (no Docker / external Postgres or Redis required).
func (c *Config) IsLocalMemory() bool {
	return strings.EqualFold(c.DatabasesMemory, "LOCAL")
}

func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8080"),
		Env:                getEnv("ENV", "development"),
		DatabasesMemory:    getEnv("DATABASES_MEMORY", ""),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://eol:eoldev@localhost:5432/eol?sslmode=disable"),
		RedisURL:           getEnv("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:          getEnv("JWT_SECRET", "dev-secret-change-me"),
		JWTAccessTTL:       parseDuration("JWT_ACCESS_TTL", 15*time.Minute),
		RefreshTokenTTL:    parseDuration("REFRESH_TOKEN_TTL", 7*24*time.Hour),
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/api/auth/google/callback"),
		CORSAllowedOrigins: strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:3000"), ","),
	}
}

func (c *Config) IsDev() bool {
	return c.Env == "development"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
