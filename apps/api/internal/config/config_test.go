package config

import (
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// t.Setenv automatically restores the original environment state after the test completes.
	t.Setenv("APP_ENV", "")
	t.Setenv("APP_PORT", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("SESSION_SECRET", "")
	t.Setenv("WEB_ORIGIN", "")
	t.Setenv("FRONTEND_ORIGIN", "")

	cfg := Load()

	if cfg.AppEnv != "development" {
		t.Errorf("expected AppEnv to be 'development', got '%s'", cfg.AppEnv)
	}

	if cfg.AppPort != "8080" {
		t.Errorf("expected AppPort to be '8080', got '%s'", cfg.AppPort)
	}

	if cfg.DatabaseURL != "" {
		t.Errorf("expected DatabaseURL to be empty, got '%s'", cfg.DatabaseURL)
	}

	if cfg.SessionSecret != "" {
		t.Errorf("expected SessionSecret to be empty, got '%s'", cfg.SessionSecret)
	}

	if cfg.FrontendOrigin != "" {
		t.Errorf("expected FrontendOrigin to be empty, got '%s'", cfg.FrontendOrigin)
	}
}

func TestLoad_ConfiguredValues(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_PORT", "9090")
	t.Setenv("SESSION_SECRET", "a-very-long-random-secret-value-here-for-testing")
	t.Setenv("WEB_ORIGIN", "https://app.example.com")

	cfg := Load()

	if cfg.AppEnv != "production" {
		t.Errorf("expected AppEnv to be 'production', got '%s'", cfg.AppEnv)
	}

	if cfg.AppPort != "9090" {
		t.Errorf("expected AppPort to be '9090', got '%s'", cfg.AppPort)
	}

	if cfg.SessionSecret != "a-very-long-random-secret-value-here-for-testing" {
		t.Errorf("expected SessionSecret to be set, got '%s'", cfg.SessionSecret)
	}

	if cfg.FrontendOrigin != "https://app.example.com" {
		t.Errorf("expected FrontendOrigin to be 'https://app.example.com', got '%s'", cfg.FrontendOrigin)
	}
}

func TestLoad_DatabaseURL_NotSet(t *testing.T) {
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

// validSecret is long enough to satisfy the SESSION_SECRET ≥ 32 character requirement.
const validSecret = "test-secret-that-is-at-least-32-chars-long"

func TestLoad_WebOriginAlias(t *testing.T) {
	t.Setenv("WEB_ORIGIN", "http://localhost:3000")
	t.Setenv("FRONTEND_ORIGIN", "")

	cfg := Load()
	if cfg.FrontendOrigin != "http://localhost:3000" {
		t.Errorf("expected FrontendOrigin to be set from WEB_ORIGIN, got '%s'", cfg.FrontendOrigin)
	}
}

func TestLoad_FrontendOriginAlias(t *testing.T) {
	t.Setenv("WEB_ORIGIN", "")
	t.Setenv("FRONTEND_ORIGIN", "http://localhost:3000")

	cfg := Load()
	if cfg.FrontendOrigin != "http://localhost:3000" {
		t.Errorf("expected FrontendOrigin to be set from FRONTEND_ORIGIN alias, got '%s'", cfg.FrontendOrigin)
	}
}

func TestLoad_FrontendOriginTakesPrecedenceOverWebOrigin(t *testing.T) {
	t.Setenv("WEB_ORIGIN", "http://localhost:3000")
	t.Setenv("FRONTEND_ORIGIN", "http://localhost:4000")

	cfg := Load()
	// FRONTEND_ORIGIN is processed last so it wins.
	if cfg.FrontendOrigin != "http://localhost:4000" {
		t.Errorf("expected FRONTEND_ORIGIN to take precedence, got '%s'", cfg.FrontendOrigin)
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
			err := (Config{FrontendOrigin: test.origin, SessionSecret: validSecret}).Validate()
			if (err == nil) != test.valid {
				t.Fatalf("Validate error = %v; valid = %v", err, test.valid)
			}
		})
	}
}

func TestValidate_SessionSecret(t *testing.T) {
	tests := []struct {
		name   string
		secret string
		valid  bool
	}{
		{name: "exactly 32 chars", secret: "12345678901234567890123456789012", valid: true},
		{name: "more than 32 chars", secret: validSecret, valid: true},
		{name: "31 chars", secret: "1234567890123456789012345678901", valid: false},
		{name: "empty", secret: "", valid: false},
	}

	validOrigin := "http://localhost:3000"
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := (Config{FrontendOrigin: validOrigin, SessionSecret: test.secret}).Validate()
			if (err == nil) != test.valid {
				t.Fatalf("Validate(SessionSecret=%q) error = %v; valid = %v", test.secret, err, test.valid)
			}
		})
	}
}
