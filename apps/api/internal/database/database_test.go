package database

import (
	"context"
	"strings"
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

func TestSessionsSchema_ApprovedMigration(t *testing.T) {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL is not configured; skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	db, err := Connect(ctx, cfg)
	if err != nil {
		t.Fatalf("Connect returned an unexpected error: %v", err)
	}
	defer db.Close()

	t.Run("table exists", func(t *testing.T) {
		var exists bool
		if err := db.Pool.QueryRow(ctx, "SELECT to_regclass('public.sessions') IS NOT NULL").Scan(&exists); err != nil {
			t.Fatalf("Querying sessions existence failed: %v", err)
		}
		if !exists {
			t.Fatal("sessions table does not exist")
		}
	})

	t.Run("columns match migration contract", func(t *testing.T) {
		rows, err := db.Pool.Query(ctx, `
			SELECT column_name, data_type, is_nullable
			FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = 'sessions'
			ORDER BY ordinal_position
		`)
		if err != nil {
			t.Fatalf("Querying sessions columns failed: %v", err)
		}
		defer rows.Close()

		actual := map[string]struct {
			dataType string
			nullable string
		}{}
		for rows.Next() {
			var name, dataType, nullable string
			if err := rows.Scan(&name, &dataType, &nullable); err != nil {
				t.Fatalf("Scanning sessions columns failed: %v", err)
			}
			actual[name] = struct {
				dataType string
				nullable string
			}{dataType: dataType, nullable: nullable}
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("Iterating sessions columns failed: %v", err)
		}

		want := map[string]struct {
			dataType string
			nullable string
		}{
			"id":           {dataType: "uuid", nullable: "NO"},
			"user_id":      {dataType: "uuid", nullable: "NO"},
			"token_hash":   {dataType: "text", nullable: "NO"},
			"expires_at":   {dataType: "timestamp with time zone", nullable: "NO"},
			"created_at":   {dataType: "timestamp with time zone", nullable: "NO"},
			"last_seen_at": {dataType: "timestamp with time zone", nullable: "YES"},
			"revoked_at":   {dataType: "timestamp with time zone", nullable: "YES"},
		}

		for name, expected := range want {
			col, ok := actual[name]
			if !ok {
				t.Fatalf("sessions.%s column is missing", name)
			}
			if col.dataType != expected.dataType {
				t.Fatalf("sessions.%s has type %q; want %q", name, col.dataType, expected.dataType)
			}
			if col.nullable != expected.nullable {
				t.Fatalf("sessions.%s nullability is %q; want %q", name, col.nullable, expected.nullable)
			}
		}
	})

	t.Run("primary key and unique constraint", func(t *testing.T) {
		var pkCount int
		if err := db.Pool.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM pg_constraint c
			JOIN pg_class tbl ON tbl.oid = c.conrelid
			JOIN pg_attribute a ON a.attrelid = tbl.oid AND a.attnum = ANY(c.conkey)
			WHERE tbl.relname = 'sessions'
			  AND c.contype = 'p'
			  AND a.attname = 'id'
		`).Scan(&pkCount); err != nil {
			t.Fatalf("Querying sessions primary key failed: %v", err)
		}
		if pkCount != 1 {
			t.Fatalf("expected exactly one primary key on sessions.id, got %d", pkCount)
		}

		var uniqueCount int
		if err := db.Pool.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM pg_constraint c
			JOIN pg_class tbl ON tbl.oid = c.conrelid
			JOIN pg_attribute a ON a.attrelid = tbl.oid AND a.attnum = ANY(c.conkey)
			WHERE tbl.relname = 'sessions'
			  AND c.contype = 'u'
			  AND a.attname = 'token_hash'
		`).Scan(&uniqueCount); err != nil {
			t.Fatalf("Querying sessions token_hash unique constraint failed: %v", err)
		}
		if uniqueCount != 1 {
			t.Fatalf("expected exactly one unique constraint on sessions.token_hash, got %d", uniqueCount)
		}
	})

	t.Run("foreign key matches approved migration", func(t *testing.T) {
		var fkCount int
		if err := db.Pool.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM pg_constraint c
			JOIN pg_class tbl ON tbl.oid = c.conrelid
			JOIN pg_class ref ON ref.oid = c.confrelid
			JOIN pg_attribute a ON a.attrelid = tbl.oid AND a.attnum = ANY(c.conkey)
			JOIN pg_attribute ra ON ra.attrelid = ref.oid AND ra.attnum = ANY(c.confkey)
			WHERE tbl.relname = 'sessions'
			  AND c.contype = 'f'
			  AND a.attname = 'user_id'
			  AND ref.relname = 'users'
			  AND ra.attname = 'id'
			  AND c.confdeltype = 'c'
		`).Scan(&fkCount); err != nil {
			t.Fatalf("Querying sessions foreign key failed: %v", err)
		}
		if fkCount != 1 {
			t.Fatalf("expected exactly one cascade foreign key from sessions.user_id to users.id, got %d", fkCount)
		}
	})

	t.Run("required indexes exist", func(t *testing.T) {
		required := []string{"idx_sessions_user_id", "idx_sessions_expires_at", "idx_sessions_active_lookup"}
		for _, indexName := range required {
			var exists bool
			if err := db.Pool.QueryRow(ctx, `
				SELECT EXISTS (
					SELECT 1
					FROM pg_indexes
					WHERE schemaname = 'public'
					  AND tablename = 'sessions'
					  AND indexname = $1
				)
			`, indexName).Scan(&exists); err != nil {
				t.Fatalf("Querying existence of index %s failed: %v", indexName, err)
			}
			if !exists {
				t.Fatalf("index %s does not exist", indexName)
			}
		}
	})

	t.Run("active lookup predicate is revoked_at is null", func(t *testing.T) {
		var indexDef string
		if err := db.Pool.QueryRow(ctx, `
			SELECT indexdef
			FROM pg_indexes
			WHERE schemaname = 'public'
			  AND tablename = 'sessions'
			  AND indexname = 'idx_sessions_active_lookup'
		`).Scan(&indexDef); err != nil {
			t.Fatalf("Querying active lookup index definition failed: %v", err)
		}
		if !strings.Contains(indexDef, "revoked_at IS NULL") {
			t.Fatalf("active lookup index is missing revoked_at IS NULL predicate: %s", indexDef)
		}
	})
}
