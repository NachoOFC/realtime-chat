package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultJWTSecret = "dev-only-change-me"

type Config struct {
	AppEnv        string
	HTTPAddr      string
	DatabaseURL   string
	CORSOrigin    string
	JWTSecret     string
	AccessTokenTTL time.Duration
	WSTicketTTL   time.Duration
	CookieSecure  bool
	CookieSameSite string
	CookieDomain  string
}

func Load() Config {
	return Config{
		AppEnv:         getenv("APP_ENV", "development"),
		HTTPAddr:       getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:    getenv("DATABASE_URL", "postgres://chat:chat@localhost:5432/chat?sslmode=disable"),
		CORSOrigin:     getenv("CORS_ORIGIN", "http://localhost:3000"),
		JWTSecret:      getenv("JWT_SECRET", defaultJWTSecret),
		AccessTokenTTL: getduration("ACCESS_TOKEN_TTL", 24*time.Hour),
		WSTicketTTL:    getduration("WS_TICKET_TTL", 30*time.Second),
		CookieSecure:   getbool("COOKIE_SECURE", false),
		CookieSameSite: strings.ToLower(getenv("COOKIE_SAME_SITE", "lax")),
		CookieDomain:   os.Getenv("COOKIE_DOMAIN"),
	}
}

func (c Config) Validate() error {
	if c.JWTSecret == "" {
		return errors.New("JWT_SECRET es obligatorio")
	}
	if c.AppEnv == "production" && c.JWTSecret == defaultJWTSecret {
		return errors.New("JWT_SECRET no puede ser el valor por defecto en producción")
	}
	switch c.CookieSameSite {
	case "lax", "strict", "none":
	default:
		return errors.New("COOKIE_SAME_SITE debe ser lax, strict o none")
	}
	if c.CookieSameSite == "none" && !c.CookieSecure {
		return errors.New("COOKIE_SAME_SITE=none requiere COOKIE_SECURE=true")
	}
	return nil
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func getbool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getduration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
