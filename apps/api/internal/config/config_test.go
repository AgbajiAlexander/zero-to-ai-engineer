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
	t.Setenv("DATABASE_URL", "")

	cfg := Load()

	if cfg.AppEnv != "development" {
		t.Errorf("expected AppEnv to be 'development', got '%s'", cfg.AppEnv)
	}

	if cfg.AppPort != "8080" {
		t.Errorf("expected AppPort to be '8080', got '%s'", cfg.AppPort)
	}

	// When DATABASE_URL is not set, DatabaseURL should be an empty string.
	if cfg.DatabaseURL != "" {
		t.Errorf("expected DatabaseURL to be empty, got '%s'", cfg.DatabaseURL)
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

func TestLoad_DatabaseURL_NotSet(t *testing.T) {
	// When DATABASE_URL is absent, DatabaseURL must be an empty string.
	t.Setenv("DATABASE_URL", "")

	cfg := Load()

	if cfg.DatabaseURL != "" {
		t.Errorf("expected DatabaseURL to be empty when not set, got '%s'", cfg.DatabaseURL)
	}
}

func TestLoad_DatabaseURL_Configured(t *testing.T) {
	// Use a placeholder DSN — no real credentials in source code.
	const fakeDSN = "postgres://app_user:placeholder@localhost:5432/zero_to_ai_engineer?sslmode=disable"
	t.Setenv("DATABASE_URL", fakeDSN)

	cfg := Load()

	if cfg.DatabaseURL != fakeDSN {
		t.Errorf("expected DatabaseURL to be '%s', got '%s'", fakeDSN, cfg.DatabaseURL)
	}
}

func TestValidate_FrontendOrigin(t *testing.T) {
	tests := []struct {
		name   string
		origin string
		valid  bool
	}{
		{name: "valid https origin", origin: "https://frontend.example", valid: true},
		{name: "valid http origin", origin: "http://localhost:3000", valid: true},
		{name: "missing", origin: "", valid: false},
		{name: "relative", origin: "/frontend", valid: false},
		{name: "path", origin: "https://frontend.example/app", valid: false},
		{name: "query", origin: "https://frontend.example?x=1", valid: false},
		{name: "unsupported scheme", origin: "ftp://frontend.example", valid: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := (Config{FrontendOrigin: test.origin}).Validate()
			if (err == nil) != test.valid {
				t.Fatalf("Validate error = %v; valid = %v", err, test.valid)
			}
		})
	}
}
