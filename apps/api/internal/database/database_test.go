package database

import (
	"context"
	"testing"
	"time"

	"github.com/zero-to-ai-engineer/api/internal/config"
)

func TestConnect_Success_UsesConfiguredDatabaseURL(t *testing.T) {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL is not configured; skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := Connect(ctx, cfg)
	if err != nil {
		t.Fatalf("Connect returned an unexpected error: %v", err)
	}
	if db == nil {
		t.Fatal("Connect returned a nil DB")
	}
	if db.Pool == nil {
		t.Fatal("Connect returned a nil pool")
	}

	db.Close()
}

func TestConnect_InvalidConfiguration_ReturnsError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := config.Config{
		DatabaseURL: "postgres://user:bad-password@127.0.0.1:1/does_not_exist?sslmode=disable",
	}

	db, err := Connect(ctx, cfg)
	if err == nil {
		t.Fatal("Connect returned nil error for an intentionally invalid database configuration")
	}
	if db != nil {
		t.Fatal("Connect returned a DB for an invalid configuration")
	}
}
