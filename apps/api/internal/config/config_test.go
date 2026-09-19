package config

import (
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Set the environment variables to empty strings to trigger the default values.
	// t.Setenv automatically restores the original environment state after the test completes,
	// ensuring tests do not affect one another.
	t.Setenv("APP_ENV", "")
	t.Setenv("APP_PORT", "")

	cfg := Load()

	if cfg.AppEnv != "development" {
		t.Errorf("expected AppEnv to be 'development', got '%s'", cfg.AppEnv)
	}

	if cfg.AppPort != "8080" {
		t.Errorf("expected AppPort to be '8080', got '%s'", cfg.AppPort)
	}
}

func TestLoad_ConfiguredValues(t *testing.T) {
	// Set custom environment variables.
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_PORT", "9090")

	cfg := Load()

	if cfg.AppEnv != "production" {
		t.Errorf("expected AppEnv to be 'production', got '%s'", cfg.AppEnv)
	}

	if cfg.AppPort != "9090" {
		t.Errorf("expected AppPort to be '9090', got '%s'", cfg.AppPort)
	}
}
