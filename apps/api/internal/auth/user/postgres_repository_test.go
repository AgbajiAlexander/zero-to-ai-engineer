package user

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

func TestPostgresRepository(t *testing.T) {
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

	repository, err := NewPostgresRepository(db)
	if err != nil {
		t.Fatalf("NewPostgresRepository returned an unexpected error: %v", err)
	}

	id := testUUID(t)
	email := fmt.Sprintf("user-repository-%s@example.com", id[:8])
	now := time.Now().UTC().Truncate(time.Microsecond)
	created := User{
		ID:           id,
		Email:        email,
		PasswordHash: "argon2id-test-hash",
		Role:         "LEARNER",
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := repository.CreateUser(ctx, created); err != nil {
		t.Fatalf("CreateUser returned an unexpected error: %v", err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, cleanupErr := db.Pool.Exec(cleanupCtx, "DELETE FROM users WHERE id = $1", id); cleanupErr != nil {
			t.Errorf("cleanup failed: %v", cleanupErr)
		}
	}()

	byEmail, err := repository.FindUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("FindUserByEmail returned an unexpected error: %v", err)
	}
	if byEmail.ID != created.ID || byEmail.Email != created.Email || byEmail.PasswordHash != created.PasswordHash || byEmail.Role != created.Role || !byEmail.IsActive {
		t.Fatalf("FindUserByEmail returned %#v; want %#v", byEmail, created)
	}

	byID, err := repository.FindUserByID(ctx, id)
	if err != nil {
		t.Fatalf("FindUserByID returned an unexpected error: %v", err)
	}
	if byID.ID != created.ID || byID.Email != created.Email {
		t.Fatalf("FindUserByID returned %#v; want %#v", byID, created)
	}

	if _, err := repository.FindUserByEmail(ctx, "missing@example.com"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("unknown email error = %v; want ErrUserNotFound", err)
	}
	if _, err := repository.FindUserByID(ctx, "not-a-uuid"); !errors.Is(err, ErrInvalidUser) {
		t.Fatalf("invalid ID error = %v; want ErrInvalidUser", err)
	}

	duplicate := created
	duplicate.ID = testUUID(t)
	if err := repository.CreateUser(ctx, duplicate); !errors.Is(err, ErrDuplicateEmail) {
		t.Fatalf("duplicate email error = %v; want ErrDuplicateEmail", err)
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
