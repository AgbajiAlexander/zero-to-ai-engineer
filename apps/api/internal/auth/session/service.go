package session

import (
	"context"
	"errors"
	"time"
)

var (
	ErrSessionExpired     = errors.New("session: session expired")
	ErrSessionRevoked     = errors.New("session: session revoked")
	ErrInvalidCreateInput = errors.New("session: invalid create input")
	ErrInvalidService     = errors.New("session: invalid service")
)

// Service owns session authentication policy above the persistence repository.
type Service interface {
	CreateSession(ctx context.Context, input CreateSessionInput) (CreateSessionResult, error)
	AuthenticateSession(ctx context.Context, rawToken string) (Session, error)
	RevokeSession(ctx context.Context, sessionID string) error
	RevokeAllUserSessions(ctx context.Context, userID string) (int, error)
}

// CreateSessionInput contains the caller-provided identity and expiry for a new session.
type CreateSessionInput struct {
	SessionID string
	UserID    string
	ExpiresAt time.Time
}

// CreateSessionResult contains the persisted session and the raw token returned once to the caller.
type CreateSessionResult struct {
	Session  Session
	RawToken string
}

// Clock supplies the current time used by session policy.
type Clock interface {
	Now() time.Time
}

type service struct {
	repository Repository
	clock      Clock
}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now()
}

var _ Service = (*service)(nil)

// NewService creates a session service using the supplied repository and clock.
// A nil clock uses the production system clock.
func NewService(repository Repository, clock Clock) (Service, error) {
	if repository == nil {
		return nil, ErrInvalidService
	}
	if clock == nil {
		clock = systemClock{}
	}
	return &service{repository: repository, clock: clock}, nil
}

func (s *service) CreateSession(ctx context.Context, input CreateSessionInput) (CreateSessionResult, error) {
	now := s.clock.Now()
	if input.SessionID == "" || input.UserID == "" || input.ExpiresAt.IsZero() || !input.ExpiresAt.After(now) {
		return CreateSessionResult{}, ErrInvalidCreateInput
	}

	rawToken, err := GenerateToken()
	if err != nil {
		return CreateSessionResult{}, err
	}
	tokenHash, err := HashToken(rawToken)
	if err != nil {
		return CreateSessionResult{}, err
	}

	created := Session{
		ID:        input.SessionID,
		UserID:    input.UserID,
		TokenHash: tokenHash,
		ExpiresAt: input.ExpiresAt,
		CreatedAt: now,
	}
	if err := s.repository.CreateSession(ctx, created); err != nil {
		return CreateSessionResult{}, err
	}

	return CreateSessionResult{Session: created, RawToken: rawToken}, nil
}

func (s *service) AuthenticateSession(ctx context.Context, rawToken string) (Session, error) {
	tokenHash, err := HashToken(rawToken)
	if err != nil {
		return Session{}, err
	}

	stored, err := s.repository.FindSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		return Session{}, err
	}
	if stored.RevokedAt != nil {
		return Session{}, ErrSessionRevoked
	}
	if !stored.ExpiresAt.After(s.clock.Now()) {
		return Session{}, ErrSessionExpired
	}
	return stored, nil
}

func (s *service) RevokeSession(ctx context.Context, sessionID string) error {
	return s.repository.RevokeSession(ctx, sessionID, s.clock.Now())
}

func (s *service) RevokeAllUserSessions(ctx context.Context, userID string) (int, error) {
	return s.repository.RevokeAllUserSessions(ctx, userID, s.clock.Now())
}
