package config

import (
	"crypto/tls"
	"errors"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultJWTSecret = "dev-only-change-me"

type Config struct {
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	FrontendURL        string
	TLSCertFile        string
	TLSKeyFile         string
	AppEnv             string
	HTTPAddr           string
	DatabaseURL        string
	CORSOrigin         string
	JWTSecret          string
	AccessTokenTTL     time.Duration
	WSTicketTTL        time.Duration
	CookieSecure       bool
	CookieSameSite     string
	CookieDomain       string
}

func Load() Config {
	return Config{
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		FrontendURL:        strings.TrimRight(getenv("FRONTEND_URL", getenv("CORS_ORIGIN", "http://localhost:3000")), "/"),
		TLSCertFile:        os.Getenv("TLS_CERT_FILE"),
		TLSKeyFile:         os.Getenv("TLS_KEY_FILE"),
		AppEnv:             getenv("APP_ENV", "development"),
		HTTPAddr:           getenv("HTTP_ADDR", ":"+getenv("PORT", "8080")),
		DatabaseURL:        getenv("DATABASE_URL", "postgres://chat:chat@localhost:5432/chat?sslmode=disable"),
		CORSOrigin:         getenv("CORS_ORIGIN", "http://localhost:3000"),
		JWTSecret:          getenv("JWT_SECRET", defaultJWTSecret),
		AccessTokenTTL:     getduration("ACCESS_TOKEN_TTL", 24*time.Hour),
		WSTicketTTL:        getduration("WS_TICKET_TTL", 30*time.Second),
		CookieSecure:       getbool("COOKIE_SECURE", false),
		CookieSameSite:     strings.ToLower(getenv("COOKIE_SAME_SITE", "lax")),
		CookieDomain:       os.Getenv("COOKIE_DOMAIN"),
	}
}

func (c Config) Validate() error {
	if c.GoogleClientID != "" || c.GoogleClientSecret != "" || c.GoogleRedirectURL != "" {
		if !c.GoogleEnabled() {
			return errors.New("configura GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET y GOOGLE_REDIRECT_URL juntos")
		}
		for _, value := range []string{c.GoogleRedirectURL, c.FrontendURL} {
			u, err := url.Parse(value)
			if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" || u.RawQuery != "" || (u.Scheme != "https" && !(c.AppEnv != "production" && u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1"))) {
				return errors.New("Google requiere URLs HTTPS válidas; HTTP solo se permite en localhost durante desarrollo")
			}
		}
		origin, _ := url.Parse(c.FrontendURL)
		if origin.Path != "" {
			return errors.New("FRONTEND_URL debe ser un origen sin rutas")
		}
	}
	if (c.TLSCertFile == "") != (c.TLSKeyFile == "") {
		return errors.New("TLS_CERT_FILE y TLS_KEY_FILE deben configurarse juntos")
	}
	if c.TLSCertFile != "" {
		if _, err := tls.LoadX509KeyPair(c.TLSCertFile, c.TLSKeyFile); err != nil {
			return errors.New("no se pudo cargar el certificado o la clave TLS")
		}
		if !c.CookieSecure {
			return errors.New("TLS requiere COOKIE_SECURE=true")
		}
	}
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

func (c Config) TLSConfig() *tls.Config { return &tls.Config{MinVersion: tls.VersionTLS12} }

func (c Config) GoogleEnabled() bool {
	return c.GoogleClientID != "" && c.GoogleClientSecret != "" && c.GoogleRedirectURL != ""
}
