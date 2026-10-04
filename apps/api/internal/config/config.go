package config

import (
	"errors"
	"net/url"
	"os"
)

// Config holds the application configuration.
type Config struct {
	AppEnv         string
	AppPort        string
	DatabaseURL    string
	SessionSecret  string
	FrontendOrigin string
}

// Load reads environment variables and returns a Config instance with defaults applied.
func Load() Config {
	cfg := Config{
		AppEnv:  "development",
		AppPort: "8080",
	}

	if env := os.Getenv("APP_ENV"); env != "" {
		cfg.AppEnv = env
	}

	if port := os.Getenv("APP_PORT"); port != "" {
		cfg.AppPort = port
	}

	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		cfg.DatabaseURL = dbURL
	}

	if secret := os.Getenv("SESSION_SECRET"); secret != "" {
		cfg.SessionSecret = secret
	}

	// WEB_ORIGIN is the canonical name used in .env / docker-compose.
	// FRONTEND_ORIGIN is accepted as an alias for backwards compatibility.
	if origin := os.Getenv("WEB_ORIGIN"); origin != "" {
		cfg.FrontendOrigin = origin
	}
	if origin := os.Getenv("FRONTEND_ORIGIN"); origin != "" {
		cfg.FrontendOrigin = origin
	}

	return cfg
}

// Validate checks configuration required by the HTTP authentication boundary.
func (cfg Config) Validate() error {
	if cfg.FrontendOrigin == "" {
		return errors.New("config: WEB_ORIGIN is required")
	}

	parsed, err := url.Parse(cfg.FrontendOrigin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("config: WEB_ORIGIN must be an absolute HTTP(S) origin")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("config: WEB_ORIGIN must use http or https")
	}

	if len(cfg.SessionSecret) < 32 {
		return errors.New("config: SESSION_SECRET must be at least 32 characters")
	}

	return nil
}
