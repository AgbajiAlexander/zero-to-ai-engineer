package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/zero-to-ai-engineer/api/internal/config"
	"github.com/zero-to-ai-engineer/api/internal/database"
)

func TestPostgresSessionRepository(t *testing.T) {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL is not configured; skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg)
	if err != nil {
		t.Fatalf("Connect returned an unexpected error: %v", err)
	}
	defer db.Close()

	repository, err := NewPostgresSessionRepository(db)
	if err != nil {
		t.Fatalf("NewPostgresSessionRepository returned an unexpected error: %v", err)
	}

	t.Run("create and find session", func(t *testing.T) {
		userID := createTestUser(t, db, ctx)
		created := testSession(t, userID, "create-find")
		if err := repository.CreateSession(ctx, created); err != nil {
			t.Fatalf("CreateSession returned an unexpected error: %v", err)
		}

		found, err := repository.FindSessionByTokenHash(ctx, created.TokenHash)
		if err != nil {
			t.Fatalf("FindSessionByTokenHash returned an unexpected error: %v", err)
		}
		assertSessionEqual(t, created, found)
	})

	t.Run("unknown token hash returns not found", func(t *testing.T) {
		_, err := repository.FindSessionByTokenHash(ctx, "unknown-token-hash")
		if !errors.Is(err, ErrSessionNotFound) {
			t.Fatalf("FindSessionByTokenHash error = %v; want ErrSessionNotFound", err)
		}
	})

	t.Run("revoke session", func(t *testing.T) {
		userID := createTestUser(t, db, ctx)
		created := testSession(t, userID, "revoke")
		if err := repository.CreateSession(ctx, created); err != nil {
			t.Fatalf("CreateSession returned an unexpected error: %v", err)
		}
		revokedAt := time.Now().UTC().Truncate(time.Microsecond)
		if err := repository.RevokeSession(ctx, created.ID, revokedAt); err != nil {
			t.Fatalf("RevokeSession returned an unexpected error: %v", err)
		}
		if err := repository.RevokeSession(ctx, created.ID, revokedAt.Add(time.Minute)); !errors.Is(err, ErrSessionAlreadyRevoked) {
			t.Fatalf("second RevokeSession error = %v; want ErrSessionAlreadyRevoked", err)
		}

		found, err := repository.FindSessionByTokenHash(ctx, created.TokenHash)
		if err != nil {
			t.Fatalf("FindSessionByTokenHash returned an unexpected error: %v", err)
		}
		if found.RevokedAt == nil || !found.RevokedAt.Equal(revokedAt) {
			t.Fatalf("retrieved revoked_at = %v; want %v", found.RevokedAt, revokedAt)
		}
	})

	t.Run("revoke all active sessions", func(t *testing.T) {
		userID := createTestUser(t, db, ctx)
		activeOne := testSession(t, userID, "revoke-all-one")
		activeTwo := testSession(t, userID, "revoke-all-two")
		alreadyRevoked := testSession(t, userID, "revoke-all-revoked")
		for _, value := range []Session{activeOne, activeTwo, alreadyRevoked} {
			if err := repository.CreateSession(ctx, value); err != nil {
				t.Fatalf("CreateSession returned an unexpected error: %v", err)
			}
		}
		originalRevokedAt := time.Now().UTC().Truncate(time.Microsecond)
		if err := repository.RevokeSession(ctx, alreadyRevoked.ID, originalRevokedAt); err != nil {
			t.Fatalf("pre-revoking session returned an unexpected error: %v", err)
		}

		revokedAt := originalRevokedAt.Add(time.Minute)
		count, err := repository.RevokeAllUserSessions(ctx, userID, revokedAt)
		if err != nil {
			t.Fatalf("RevokeAllUserSessions returned an unexpected error: %v", err)
		}
		if count != 2 {
			t.Fatalf("RevokeAllUserSessions changed %d sessions; want 2", count)
		}

		found, err := repository.FindSessionByTokenHash(ctx, alreadyRevoked.TokenHash)
		if err != nil {
			t.Fatalf("FindSessionByTokenHash returned an unexpected error: %v", err)
		}
		if found.RevokedAt == nil || !found.RevokedAt.Equal(originalRevokedAt) {
			t.Fatalf("already revoked timestamp = %v; want %v", found.RevokedAt, originalRevokedAt)
		}
	})

	t.Run("expired and revoked sessions remain retrievable", func(t *testing.T) {
		userID := createTestUser(t, db, ctx)
		expired := testSession(t, userID, "expired")
		expired.ExpiresAt = time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
		if err := repository.CreateSession(ctx, expired); err != nil {
			t.Fatalf("CreateSession returned an unexpected error: %v", err)
		}
		foundExpired, err := repository.FindSessionByTokenHash(ctx, expired.TokenHash)
		if err != nil {
			t.Fatalf("FindSessionByTokenHash returned an unexpected error: %v", err)
		}
		if !foundExpired.ExpiresAt.Equal(expired.ExpiresAt) {
			t.Fatalf("expired session expiration = %v; want %v", foundExpired.ExpiresAt, expired.ExpiresAt)
		}

		revoked := testSession(t, userID, "retrievable-revoked")
		if err := repository.CreateSession(ctx, revoked); err != nil {
			t.Fatalf("CreateSession returned an unexpected error: %v", err)
		}
		if err := repository.RevokeSession(ctx, revoked.ID, time.Now().UTC()); err != nil {
			t.Fatalf("RevokeSession returned an unexpected error: %v", err)
		}
		if _, err := repository.FindSessionByTokenHash(ctx, revoked.TokenHash); err != nil {
			t.Fatalf("FindSessionByTokenHash rejected revoked session: %v", err)
		}
	})

	t.Run("context cancellation is respected", func(t *testing.T) {
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		_, err := repository.FindSessionByTokenHash(canceled, "context-canceled-token")
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("FindSessionByTokenHash error = %v; want context.Canceled", err)
		}
	})

	t.Run("foreign key is enforced", func(t *testing.T) {
		orphan := testSession(t, testUUID(t), "orphan")
		if err := repository.CreateSession(ctx, orphan); err == nil {
			t.Fatal("CreateSession succeeded for a session with no matching user")
		}
	})

	t.Run("duplicate token hashes are rejected", func(t *testing.T) {
		userID := createTestUser(t, db, ctx)
		first := testSession(t, userID, "duplicate-first")
		second := testSession(t, userID, "duplicate-second")
		second.TokenHash = first.TokenHash
		if err := repository.CreateSession(ctx, first); err != nil {
			t.Fatalf("CreateSession for first session returned an unexpected error: %v", err)
		}
		if err := repository.CreateSession(ctx, second); err == nil {
			t.Fatal("CreateSession accepted a duplicate token hash")
		}
	})
}

func testSession(t *testing.T, userID, label string) Session {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	tokenHash, err := HashToken(label + "-" + testUUID(t))
	if err != nil {
		t.Fatalf("HashToken returned an unexpected error: %v", err)
	}
	return Session{
		ID:        testUUID(t),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
	}
}

func createTestUser(t *testing.T, db *database.DB, ctx context.Context) string {
	t.Helper()
	userID := testUUID(t)
	email := fmt.Sprintf("session-repository-%s@example.com", userID[:8])
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash)
		VALUES ($1, $2, $3)
	`, userID, email, "test-password-hash"); err != nil {
		t.Fatalf("inserting isolated test user failed: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, err := db.Pool.Exec(cleanupCtx, "DELETE FROM users WHERE id = $1", userID); err != nil {
			t.Errorf("cleaning up isolated test user failed: %v", err)
		}
	})
	return userID
}

func assertSessionEqual(t *testing.T, want, got Session) {
	t.Helper()
	if got.ID != want.ID || got.UserID != want.UserID || got.TokenHash != want.TokenHash ||
		!got.ExpiresAt.Equal(want.ExpiresAt) || !got.CreatedAt.Equal(want.CreatedAt) ||
		got.LastSeenAt != nil || got.RevokedAt != nil {
		t.Fatalf("retrieved session = %#v; want %#v", got, want)
	}
}

func testUUID(t *testing.T) string {
	t.Helper()
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		t.Fatalf("generating test UUID failed: %v", err)
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(value)
	return encoded[:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:]
}
