package session

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type fakeRepository struct {
	createdSession   Session
	createCalls      int
	createContext    context.Context
	createErr        error
	foundSession     Session
	findCalls        int
	findContext      context.Context
	findTokenHash    string
	findErr          error
	revokeSessionID  string
	revokeAt         time.Time
	revokeContext    context.Context
	revokeErr        error
	revokeAllUserID  string
	revokeAllAt      time.Time
	revokeAllContext context.Context
	revokeAllCount   int
	revokeAllErr     error
}

func (f *fakeRepository) CreateSession(ctx context.Context, value Session) error {
	f.createCalls++
	f.createContext = ctx
	if f.createErr != nil {
		return f.createErr
	}
	f.createdSession = value
	return nil
}

func (f *fakeRepository) FindSessionByTokenHash(ctx context.Context, tokenHash string) (Session, error) {
	f.findCalls++
	f.findContext = ctx
	f.findTokenHash = tokenHash
	if f.findErr != nil {
		return Session{}, f.findErr
	}
	return f.foundSession, nil
}

func (f *fakeRepository) RevokeSession(ctx context.Context, sessionID string, revokedAt time.Time) error {
	f.revokeSessionID = sessionID
	f.revokeAt = revokedAt
	f.revokeContext = ctx
	return f.revokeErr
}

func (f *fakeRepository) RevokeAllUserSessions(ctx context.Context, userID string, revokedAt time.Time) (int, error) {
	f.revokeAllUserID = userID
	f.revokeAllAt = revokedAt
	f.revokeAllContext = ctx
	return f.revokeAllCount, f.revokeAllErr
}

func newTestService(t *testing.T, repository Repository, now time.Time) Service {
	t.Helper()
	service, err := NewService(repository, fixedClock{now: now})
	if err != nil {
		t.Fatalf("NewService returned an unexpected error: %v", err)
	}
	return service
}

func TestCreateSession_SuccessfullyGeneratesAndPersistsHashedToken(t *testing.T) {
	now := time.Date(2026, 9, 21, 20, 0, 0, 0, time.UTC)
	repository := &fakeRepository{}
	service := newTestService(t, repository, now)
	input := CreateSessionInput{
		SessionID: "session-id",
		UserID:    "user-id",
		ExpiresAt: now.Add(time.Hour),
	}
	ctx := context.WithValue(context.Background(), "request", "create")

	result, err := service.CreateSession(ctx, input)
	if err != nil {
		t.Fatalf("CreateSession returned an unexpected error: %v", err)
	}
	if result.RawToken == "" {
		t.Fatal("CreateSession returned an empty raw token")
	}
	if repository.createCalls != 1 || repository.createContext != ctx {
		t.Fatal("CreateSession did not pass the caller context to the repository")
	}
	if repository.createdSession.TokenHash == "" {
		t.Fatal("CreateSession persisted an empty token hash")
	}
	if repository.createdSession.TokenHash == result.RawToken {
		t.Fatal("CreateSession persisted the raw token")
	}
	expectedHash, err := HashToken(result.RawToken)
	if err != nil {
		t.Fatalf("HashToken returned an unexpected error: %v", err)
	}
	if repository.createdSession.TokenHash != expectedHash || result.Session.TokenHash != expectedHash {
		t.Fatal("CreateSession did not use the hash of the returned token")
	}
	if result.Session.ID != input.SessionID || result.Session.UserID != input.UserID {
		t.Fatalf("CreateSession returned unexpected identity: %#v", result.Session)
	}
	if !result.Session.CreatedAt.Equal(now) || !result.Session.ExpiresAt.Equal(input.ExpiresAt) {
		t.Fatalf("CreateSession returned unexpected timestamps: %#v", result.Session)
	}
	if result.Session.LastSeenAt != nil || result.Session.RevokedAt != nil {
		t.Fatal("CreateSession initialized nullable session timestamps")
	}
}

func TestCreateSession_RejectsInvalidInput(t *testing.T) {
	now := time.Date(2026, 9, 21, 20, 0, 0, 0, time.UTC)
	valid := CreateSessionInput{
		SessionID: "session-id",
		UserID:    "user-id",
		ExpiresAt: now.Add(time.Hour),
	}
	tests := []struct {
		name  string
		input CreateSessionInput
	}{
		{name: "empty session ID", input: CreateSessionInput{UserID: valid.UserID, ExpiresAt: valid.ExpiresAt}},
		{name: "empty user ID", input: CreateSessionInput{SessionID: valid.SessionID, ExpiresAt: valid.ExpiresAt}},
		{name: "zero expiration", input: CreateSessionInput{SessionID: valid.SessionID, UserID: valid.UserID}},
		{name: "expiration equal to now", input: CreateSessionInput{SessionID: valid.SessionID, UserID: valid.UserID, ExpiresAt: now}},
		{name: "expiration before now", input: CreateSessionInput{SessionID: valid.SessionID, UserID: valid.UserID, ExpiresAt: now.Add(-time.Second)}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &fakeRepository{}
			service := newTestService(t, repository, now)
			_, err := service.CreateSession(context.Background(), test.input)
			if !errors.Is(err, ErrInvalidCreateInput) {
				t.Fatalf("CreateSession error = %v; want ErrInvalidCreateInput", err)
			}
			if repository.createCalls != 0 {
				t.Fatal("CreateSession called the repository for invalid input")
			}
		})
	}
}

func TestCreateSession_PreservesRepositoryError(t *testing.T) {
	repositoryError := errors.New("repository unavailable")
	now := time.Date(2026, 9, 21, 20, 0, 0, 0, time.UTC)
	service := newTestService(t, &fakeRepository{createErr: repositoryError}, now)

	_, err := service.CreateSession(context.Background(), CreateSessionInput{
		SessionID: "session-id",
		UserID:    "user-id",
		ExpiresAt: now.Add(time.Hour),
	})
	if !errors.Is(err, repositoryError) {
		t.Fatalf("CreateSession error = %v; want repository error", err)
	}
}

func TestAuthenticateSession_SuccessfullyReturnsUnchangedSession(t *testing.T) {
	now := time.Date(2026, 9, 21, 20, 0, 0, 0, time.UTC)
	lastSeen := now.Add(-time.Minute)
	stored := Session{
		ID:         "session-id",
		UserID:     "user-id",
		TokenHash:  "stored-hash",
		ExpiresAt:  now.Add(time.Hour),
		CreatedAt:  now.Add(-time.Hour),
		LastSeenAt: &lastSeen,
	}
	repository := &fakeRepository{foundSession: stored}
	service := newTestService(t, repository, now)
	ctx := context.WithValue(context.Background(), "request", "authenticate")

	got, err := service.AuthenticateSession(ctx, "raw-session-token")
	if err != nil {
		t.Fatalf("AuthenticateSession returned an unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, stored) {
		t.Fatalf("AuthenticateSession returned %#v; want %#v", got, stored)
	}
	if repository.findCalls != 1 || repository.findContext != ctx {
		t.Fatal("AuthenticateSession did not pass the caller context to the repository")
	}
	expectedHash, err := HashToken("raw-session-token")
	if err != nil {
		t.Fatalf("HashToken returned an unexpected error: %v", err)
	}
	if repository.findTokenHash != expectedHash {
		t.Fatalf("lookup hash = %q; want %q", repository.findTokenHash, expectedHash)
	}
	if !reflect.DeepEqual(repository.foundSession, stored) {
		t.Fatal("AuthenticateSession mutated the repository session")
	}
}

func TestAuthenticateSession_RejectsInvalidTokensAndSessions(t *testing.T) {
	now := time.Date(2026, 9, 21, 20, 0, 0, 0, time.UTC)
	revokedAt := now.Add(-time.Minute)
	tests := []struct {
		name       string
		rawToken   string
		stored     Session
		repository error
		want       error
	}{
		{name: "empty token", want: ErrEmptyToken},
		{name: "unknown token", rawToken: "unknown", repository: ErrSessionNotFound, want: ErrSessionNotFound},
		{name: "revoked session", rawToken: "revoked", stored: Session{ExpiresAt: now.Add(time.Hour), RevokedAt: &revokedAt}, want: ErrSessionRevoked},
		{name: "expired session", rawToken: "expired", stored: Session{ExpiresAt: now.Add(-time.Second)}, want: ErrSessionExpired},
		{name: "exact expiry boundary", rawToken: "boundary", stored: Session{ExpiresAt: now}, want: ErrSessionExpired},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &fakeRepository{foundSession: test.stored, findErr: test.repository}
			service := newTestService(t, repository, now)
			_, err := service.AuthenticateSession(context.Background(), test.rawToken)
			if !errors.Is(err, test.want) {
				t.Fatalf("AuthenticateSession error = %v; want %v", err, test.want)
			}
		})
	}
}

func TestRevokeSession_DelegatesTimestampAndPreservesErrors(t *testing.T) {
	now := time.Date(2026, 9, 21, 20, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		err  error
	}{
		{name: "success"},
		{name: "not found", err: ErrSessionNotFound},
		{name: "already revoked", err: ErrSessionAlreadyRevoked},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &fakeRepository{revokeErr: test.err}
			service := newTestService(t, repository, now)
			ctx := context.WithValue(context.Background(), "request", "revoke")
			err := service.RevokeSession(ctx, "session-id")
			if !errors.Is(err, test.err) {
				t.Fatalf("RevokeSession error = %v; want %v", err, test.err)
			}
			if repository.revokeSessionID != "session-id" || !repository.revokeAt.Equal(now) || repository.revokeContext != ctx {
				t.Fatal("RevokeSession did not delegate the session ID, clock time, and context")
			}
		})
	}
}

func TestRevokeAllUserSessions_DelegatesTimestampCountAndError(t *testing.T) {
	now := time.Date(2026, 9, 21, 20, 0, 0, 0, time.UTC)
	repositoryError := errors.New("repository unavailable")
	repository := &fakeRepository{revokeAllCount: 3, revokeAllErr: repositoryError}
	service := newTestService(t, repository, now)
	ctx := context.WithValue(context.Background(), "request", "revoke-all")

	count, err := service.RevokeAllUserSessions(ctx, "user-id")
	if count != 3 || !errors.Is(err, repositoryError) {
		t.Fatalf("RevokeAllUserSessions returned count=%d, error=%v; want count=3 and repository error", count, err)
	}
	if repository.revokeAllUserID != "user-id" || !repository.revokeAllAt.Equal(now) || repository.revokeAllContext != ctx {
		t.Fatal("RevokeAllUserSessions did not delegate the user ID, clock time, and context")
	}
}
