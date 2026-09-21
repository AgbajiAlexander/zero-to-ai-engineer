package session

import (
	"context"
	"errors"
	"time"
)

var (
	ErrSessionNotFound       = errors.New("session: session not found")
	ErrSessionAlreadyRevoked = errors.New("session: already revoked")
	ErrInvalidSession        = errors.New("session: invalid session")
	ErrInvalidTokenHash      = errors.New("session: invalid token hash")
)

// Repository persists sessions without making authentication or authorisation decisions.
// Implementations must return sessions as stored, including expired or revoked sessions.
type Repository interface {
	// CreateSession persists a new session. An invalid session returns ErrInvalidSession.
	CreateSession(ctx context.Context, session Session) error

	// FindSessionByTokenHash retrieves a session by its stored token hash.
	// It returns ErrSessionNotFound when no session matches. It does not filter by
	// expiration or revocation; the service layer applies those authentication rules.
	FindSessionByTokenHash(ctx context.Context, tokenHash string) (Session, error)

	// RevokeSession records revokedAt for one session. It returns ErrSessionNotFound
	// when the session does not exist and ErrSessionAlreadyRevoked when it is revoked.
	RevokeSession(ctx context.Context, sessionID string, revokedAt time.Time) error

	// RevokeAllUserSessions records revokedAt for every active session owned by userID.
	// It returns the number of sessions changed. Expired sessions are still revoked;
	// the repository does not make authentication decisions.
	RevokeAllUserSessions(ctx context.Context, userID string, revokedAt time.Time) (int, error)
}
