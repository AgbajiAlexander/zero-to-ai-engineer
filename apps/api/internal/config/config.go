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

	if origin := os.Getenv("FRONTEND_ORIGIN"); origin != "" {
		cfg.FrontendOrigin = origin
	}

	return cfg
}

// Validate checks configuration required by the HTTP authentication boundary.
func (cfg Config) Validate() error {
	if cfg.FrontendOrigin == "" {
		return errors.New("config: FRONTEND_ORIGIN is required")
	}

	parsed, err := url.Parse(cfg.FrontendOrigin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("config: FRONTEND_ORIGIN must be an absolute HTTP(S) origin")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("config: FRONTEND_ORIGIN must use http or https")
	}
	return nil
}
