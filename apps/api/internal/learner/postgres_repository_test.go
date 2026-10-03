package learner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
	"time"

	authuser "github.com/zero-to-ai-engineer/api/internal/auth/user"
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

	repository, err := NewPostgresRepository(db.Pool)
	if err != nil {
		t.Fatalf("NewPostgresRepository returned an unexpected error: %v", err)
	}

	userRepository, err := authuser.NewPostgresRepository(db)
	if err != nil {
		t.Fatalf("creating user repository failed: %v", err)
	}

	userID := learnerTestUUID(t)
	email := fmt.Sprintf("learner-repository-%s@example.com", userID[:8])
	now := time.Now().UTC().Truncate(time.Microsecond)

	testUser := authuser.User{
		ID:           userID,
		Email:        email,
		PasswordHash: "argon2id-test-hash",
		Role:         "LEARNER",
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := userRepository.CreateUser(ctx, testUser); err != nil {
		t.Fatalf("creating test user failed: %v", err)
	}

	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cleanupCancel()

		// Deleting the user also deletes the learner profile through
		// the foreign-key cascade defined in the migration.
		if _, cleanupErr := db.Pool.Exec(
			cleanupCtx,
			"DELETE FROM users WHERE id = $1",
			userID,
		); cleanupErr != nil {
			t.Errorf("cleanup failed: %v", cleanupErr)
		}
	}()

	created := Learner{
		UserID:      userID,
		DisplayName: "Alice Engineer",
	}

	if err := repository.Create(ctx, created); err != nil {
		t.Fatalf("Create returned an unexpected error: %v", err)
	}

	got, err := repository.FindByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("FindByUserID returned an unexpected error: %v", err)
	}

	if got.UserID != created.UserID ||
		got.DisplayName != created.DisplayName {
		t.Fatalf("FindByUserID returned %#v; want %#v", got, created)
	}

	if err := repository.Create(ctx, created); !errors.Is(err, ErrLearnerExists) {
		t.Fatalf("duplicate profile error = %v; want ErrLearnerExists", err)
	}

	missingUserID := learnerTestUUID(t)
	if _, err := repository.FindByUserID(ctx, missingUserID); !errors.Is(err, ErrLearnerNotFound) {
		t.Fatalf("missing profile error = %v; want ErrLearnerNotFound", err)
	}

	unknownUserID := learnerTestUUID(t)
	err = repository.Create(ctx, Learner{
		UserID:      unknownUserID,
		DisplayName: "Unknown User",
	})
	if err == nil {
		t.Fatal("Create succeeded for a user that does not exist; want foreign-key error")
	}
	if errors.Is(err, ErrLearnerExists) {
		t.Fatalf("foreign-key error was incorrectly mapped to ErrLearnerExists: %v", err)
	}
}

func learnerTestUUID(t *testing.T) string {
	t.Helper()

	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		t.Fatalf("generating test UUID failed: %v", err)
	}

	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80

	encoded := hex.EncodeToString(value)
	return encoded[:8] + "-" +
		encoded[8:12] + "-" +
		encoded[12:16] + "-" +
		encoded[16:20] + "-" +
		encoded[20:]
}
