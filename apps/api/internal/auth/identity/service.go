package identity

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/zero-to-ai-engineer/api/internal/auth"
	"github.com/zero-to-ai-engineer/api/internal/auth/session"
	"github.com/zero-to-ai-engineer/api/internal/auth/user"
)

const (
	minPasswordLength = 12
	maxPasswordLength = 128
	sessionLifetime   = 24 * time.Hour
	defaultUserRole   = "LEARNER"
)

var (
	ErrInvalidInput       = errors.New("identity: invalid input")
	ErrInvalidCredentials = errors.New("identity: invalid credentials")
	ErrInactiveAccount    = errors.New("identity: inactive account")
	ErrRateLimited        = errors.New("identity: rate limit exceeded")
	ErrInvalidService     = errors.New("identity: invalid service")
)

type Service interface {
	Register(ctx context.Context, input RegisterInput) (user.User, error)
	Login(ctx context.Context, input LoginInput) (LoginResult, error)
	GetUser(ctx context.Context, userID string) (user.User, error)
}

type RegisterInput struct {
	Email    string
	Password string
}

type LoginInput struct {
	Email    string
	Password string
}

type LoginResult struct {
	User    user.User
	Session session.CreateSessionResult
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID() (string, error)
}

type RateLimiter interface {
	Allow(ctx context.Context, key string) error
}

type service struct {
	users      user.Repository
	sessions   session.Service
	clock      Clock
	ids        IDGenerator
	rateLimits RateLimiter
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

type cryptoIDGenerator struct{}

func (cryptoIDGenerator) NewID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("identity: generate ID: %w", err)
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(value)
	return encoded[:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:], nil
}

var _ Service = (*service)(nil)

func NewService(users user.Repository, sessions session.Service, clock Clock, ids IDGenerator, rateLimits RateLimiter) (Service, error) {
	if users == nil || sessions == nil || rateLimits == nil {
		return nil, ErrInvalidService
	}
	if clock == nil {
		clock = systemClock{}
	}
	if ids == nil {
		ids = cryptoIDGenerator{}
	}
	return &service{users: users, sessions: sessions, clock: clock, ids: ids, rateLimits: rateLimits}, nil
}

func (s *service) Register(ctx context.Context, input RegisterInput) (user.User, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil || !validPassword(input.Password) {
		return user.User{}, ErrInvalidInput
	}
	if err := s.rateLimits.Allow(ctx, "register:"+email); err != nil {
		if errors.Is(err, ErrRateLimited) {
			return user.User{}, ErrRateLimited
		}
		return user.User{}, err
	}

	id, err := s.ids.NewID()
	if err != nil {
		return user.User{}, fmt.Errorf("identity: generate user ID: %w", err)
	}
	passwordHash, err := auth.HashPassword(input.Password)
	if err != nil {
		return user.User{}, fmt.Errorf("identity: hash password: %w", err)
	}
	now := s.clock.Now()
	created := user.User{
		ID:           id,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         defaultUserRole,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.users.CreateUser(ctx, created); err != nil {
		return user.User{}, err
	}
	return created, nil
}

func (s *service) Login(ctx context.Context, input LoginInput) (LoginResult, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil || !validPassword(input.Password) {
		return LoginResult{}, ErrInvalidCredentials
	}
	if err := s.rateLimits.Allow(ctx, "login:"+email); err != nil {
		if errors.Is(err, ErrRateLimited) {
			return LoginResult{}, ErrRateLimited
		}
		return LoginResult{}, err
	}

	stored, err := s.users.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return LoginResult{}, ErrInvalidCredentials
		}
		return LoginResult{}, err
	}
	if !stored.IsActive || stored.DeletedAt != nil || !auth.VerifyPassword(input.Password, stored.PasswordHash) {
		return LoginResult{}, ErrInvalidCredentials
	}

	sessionID, err := s.ids.NewID()
	if err != nil {
		return LoginResult{}, fmt.Errorf("identity: generate session ID: %w", err)
	}
	created, err := s.sessions.CreateSession(ctx, session.CreateSessionInput{
		SessionID: sessionID,
		UserID:    stored.ID,
		ExpiresAt: s.clock.Now().Add(sessionLifetime),
	})
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{User: stored, Session: created}, nil
}

func (s *service) GetUser(ctx context.Context, userID string) (user.User, error) {
	if strings.TrimSpace(userID) == "" {
		return user.User{}, ErrInvalidInput
	}
	stored, err := s.users.FindUserByID(ctx, userID)
	if err != nil {
		return user.User{}, err
	}
	if !stored.IsActive || stored.DeletedAt != nil {
		return user.User{}, ErrInactiveAccount
	}
	return stored, nil
}

func normalizeEmail(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" || strings.ContainsAny(normalized, "\r\n") {
		return "", ErrInvalidInput
	}
	parsed, err := mail.ParseAddress(normalized)
	if err != nil || parsed.Address != normalized || !strings.Contains(normalized, "@") {
		return "", ErrInvalidInput
	}
	return normalized, nil
}

func validPassword(value string) bool {
	return utf8.ValidString(value) && utf8.RuneCountInString(value) >= minPasswordLength && utf8.RuneCountInString(value) <= maxPasswordLength
}
