package identity

import (
	"context"
	"crypto/sha256"
	"errors"
	"sync"
	"time"
)

const (
	DefaultRateLimitAttempts = 5
	DefaultRateLimitWindow   = time.Minute
	DefaultRateLimitMaxKeys  = 10000
)

var ErrInvalidRateLimiter = errors.New("identity: invalid rate limiter")

type rateLimitEntry struct {
	windowStart time.Time
	attempts    int
	lastSeen    time.Time
}

// MemoryRateLimiter is a bounded, concurrency-safe initial limiter for identity operations.
// It stores only SHA-256 key digests, never emails, passwords, tokens, or timestamps supplied by clients.
type MemoryRateLimiter struct {
	mu          sync.Mutex
	entries     map[[32]byte]rateLimitEntry
	maxAttempts int
	window      time.Duration
	maxKeys     int
	now         func() time.Time
}

var _ RateLimiter = (*MemoryRateLimiter)(nil)

// NewMemoryRateLimiter creates a bounded fixed-window limiter.
// Each key is allowed maxAttempts requests during window; expired entries are cleaned
// on every decision, and the oldest entry is evicted when maxKeys is reached.
func NewMemoryRateLimiter(maxAttempts int, window time.Duration, maxKeys int) (*MemoryRateLimiter, error) {
	if maxAttempts <= 0 || window <= 0 || maxKeys <= 0 {
		return nil, ErrInvalidRateLimiter
	}
	return &MemoryRateLimiter{
		entries:     make(map[[32]byte]rateLimitEntry),
		maxAttempts: maxAttempts,
		window:      window,
		maxKeys:     maxKeys,
		now:         time.Now,
	}, nil
}

func (l *MemoryRateLimiter) Allow(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	keyDigest := sha256.Sum256([]byte(key))
	now := l.now()

	l.mu.Lock()
	defer l.mu.Unlock()

	l.cleanupExpired(now)
	entry, exists := l.entries[keyDigest]
	if !exists {
		if len(l.entries) >= l.maxKeys {
			l.evictOldest()
		}
		l.entries[keyDigest] = rateLimitEntry{windowStart: now, attempts: 1, lastSeen: now}
		return nil
	}

	if !now.Before(entry.windowStart.Add(l.window)) {
		l.entries[keyDigest] = rateLimitEntry{windowStart: now, attempts: 1, lastSeen: now}
		return nil
	}
	entry.lastSeen = now
	if entry.attempts >= l.maxAttempts {
		l.entries[keyDigest] = entry
		return ErrRateLimited
	}
	entry.attempts++
	l.entries[keyDigest] = entry
	return nil
}

func (l *MemoryRateLimiter) cleanupExpired(now time.Time) {
	for key, entry := range l.entries {
		if !now.Before(entry.windowStart.Add(l.window)) {
			delete(l.entries, key)
		}
	}
}

func (l *MemoryRateLimiter) evictOldest() {
	var oldestKey [32]byte
	var oldest time.Time
	first := true
	for key, entry := range l.entries {
		if first || entry.lastSeen.Before(oldest) {
			oldestKey = key
			oldest = entry.lastSeen
			first = false
		}
	}
	if !first {
		delete(l.entries, oldestKey)
	}
}
