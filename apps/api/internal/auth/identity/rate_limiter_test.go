package identity

import (
	"context"
	"crypto/sha256"
	"errors"
	"sync"
	"testing"
	"time"
)

func newTestLimiter(t *testing.T, attempts, maxKeys int, now *time.Time) *MemoryRateLimiter {
	t.Helper()
	limiter, err := NewMemoryRateLimiter(attempts, time.Minute, maxKeys)
	if err != nil {
		t.Fatalf("NewMemoryRateLimiter returned an unexpected error: %v", err)
	}
	limiter.now = func() time.Time { return *now }
	return limiter
}

func TestMemoryRateLimiter_AllowsRequestsBelowLimit(t *testing.T) {
	now := time.Date(2026, 9, 21, 20, 0, 0, 0, time.UTC)
	limiter := newTestLimiter(t, 3, 10, &now)
	for i := 0; i < 3; i++ {
		if err := limiter.Allow(context.Background(), "login:user@example.com"); err != nil {
			t.Fatalf("Allow request %d returned an unexpected error: %v", i+1, err)
		}
	}
}

func TestMemoryRateLimiter_EnforcesLimit(t *testing.T) {
	now := time.Date(2026, 9, 21, 20, 0, 0, 0, time.UTC)
	limiter := newTestLimiter(t, 2, 10, &now)
	if err := limiter.Allow(context.Background(), "login:user@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := limiter.Allow(context.Background(), "login:user@example.com"); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(limiter.Allow(context.Background(), "login:user@example.com"), ErrRateLimited) {
		t.Fatal("third request was not rate limited")
	}
}

func TestMemoryRateLimiter_WindowExpirationAllowsRequestsAgain(t *testing.T) {
	now := time.Date(2026, 9, 21, 20, 0, 0, 0, time.UTC)
	limiter := newTestLimiter(t, 1, 10, &now)
	key := "register:user@example.com"
	if err := limiter.Allow(context.Background(), key); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(limiter.Allow(context.Background(), key), ErrRateLimited) {
		t.Fatal("request was not limited before window expiration")
	}
	now = now.Add(time.Minute)
	if err := limiter.Allow(context.Background(), key); err != nil {
		t.Fatalf("request after window expiration returned an unexpected error: %v", err)
	}
}

func TestMemoryRateLimiter_ConcurrentAccessIsSafe(t *testing.T) {
	now := time.Date(2026, 9, 21, 20, 0, 0, 0, time.UTC)
	limiter := newTestLimiter(t, 100, 10, &now)
	const callers = 50
	var wait sync.WaitGroup
	wait.Add(callers)
	for i := 0; i < callers; i++ {
		go func() {
			defer wait.Done()
			_ = limiter.Allow(context.Background(), "login:user@example.com")
		}()
	}
	wait.Wait()
	if limiter.entries == nil {
		t.Fatal("limiter state was not initialized")
	}
}

func TestMemoryRateLimiter_SeparateKeysAreIndependent(t *testing.T) {
	now := time.Date(2026, 9, 21, 20, 0, 0, 0, time.UTC)
	limiter := newTestLimiter(t, 1, 10, &now)
	if err := limiter.Allow(context.Background(), "login:first@example.com"); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(limiter.Allow(context.Background(), "login:first@example.com"), ErrRateLimited) {
		t.Fatal("first key was not limited")
	}
	if err := limiter.Allow(context.Background(), "login:second@example.com"); err != nil {
		t.Fatalf("second key was incorrectly limited: %v", err)
	}
}

func TestMemoryRateLimiter_BoundsAndCleansState(t *testing.T) {
	now := time.Date(2026, 9, 21, 20, 0, 0, 0, time.UTC)
	limiter := newTestLimiter(t, 1, 2, &now)
	for _, key := range []string{"login:first@example.com", "login:second@example.com", "login:third@example.com"} {
		if err := limiter.Allow(context.Background(), key); err != nil {
			t.Fatal(err)
		}
	}
	if len(limiter.entries) != 2 {
		t.Fatalf("entry count = %d; want bounded count 2", len(limiter.entries))
	}
	now = now.Add(time.Minute)
	if err := limiter.Allow(context.Background(), "login:new@example.com"); err != nil {
		t.Fatal(err)
	}
	if len(limiter.entries) != 1 {
		t.Fatalf("entry count after expiration cleanup = %d; want 1", len(limiter.entries))
	}
}

func TestMemoryRateLimiter_DoesNotStorePlaintextCredentials(t *testing.T) {
	now := time.Date(2026, 9, 21, 20, 0, 0, 0, time.UTC)
	limiter := newTestLimiter(t, 1, 10, &now)
	key := "login:user@example.com"
	if err := limiter.Allow(context.Background(), key); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(key))
	if _, ok := limiter.entries[digest]; !ok {
		t.Fatal("limiter did not retain the hashed key")
	}
	for stored := range limiter.entries {
		if string(stored[:]) == key || string(stored[:]) == "password" || string(stored[:]) == "raw-token" {
			t.Fatal("limiter stored plaintext credential data")
		}
	}
}

func TestMemoryRateLimiter_RespectsCanceledContext(t *testing.T) {
	limiter, err := NewMemoryRateLimiter(1, time.Minute, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !errors.Is(limiter.Allow(ctx, "login:user@example.com"), context.Canceled) {
		t.Fatal("canceled context was not respected")
	}
}
