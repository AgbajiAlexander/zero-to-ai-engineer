package config

import "os"

// Config holds the application configuration.
type Config struct {
	AppEnv  string
	AppPort string
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

	return cfg
}
