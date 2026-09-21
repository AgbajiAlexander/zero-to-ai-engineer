package session

import "time"

// Session is the persisted identity of an authenticated session.
// TokenHash is safe to persist; the raw session token is intentionally absent.
type Session struct {
	ID         string
	UserID     string
	TokenHash  string
	ExpiresAt  time.Time
	CreatedAt  time.Time
	LastSeenAt *time.Time
	RevokedAt  *time.Time
}
