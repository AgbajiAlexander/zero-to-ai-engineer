package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zero-to-ai-engineer/api/internal/config"
)

// DB wraps a pgxpool.Pool and provides a handle to the PostgreSQL connection pool.
// It belongs to the infrastructure layer and must not contain domain or business logic.
type DB struct {
	Pool *pgxpool.Pool
}

// Connect creates a new PostgreSQL connection pool using the provided configuration,
// verifies connectivity with a Ping, and returns a DB instance.
// It returns an error if the pool cannot be created or if Ping fails.
func Connect(ctx context.Context, cfg config.Config) (*DB, error) {
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("database: DATABASE_URL is not set")
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("database: failed to create connection pool: %w", err)
	}

	// Verify the database is reachable before returning.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database: failed to reach PostgreSQL: %w", err)
	}

	return &DB{Pool: pool}, nil
}

// Close safely shuts down the connection pool.
// It should be deferred after a successful Connect call.
func (db *DB) Close() {
	db.Pool.Close()
}
