package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/zero-to-ai-engineer/api/internal/auth"
	"github.com/zero-to-ai-engineer/api/internal/auth/session"
	"github.com/zero-to-ai-engineer/api/internal/auth/user"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type fixedIDs struct {
	values []string
}

func (g *fixedIDs) NewID() (string, error) {
	value := g.values[0]
	g.values = g.values[1:]
	return value, nil
}

type fakeUserRepository struct {
	created   user.User
	createErr error
	byEmail   user.User
	emailErr  error
	byID      user.User
	idErr     error
}

func (f *fakeUserRepository) CreateUser(_ context.Context, value user.User) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.created = value
	return nil
}

func (f *fakeUserRepository) FindUserByEmail(context.Context, string) (user.User, error) {
	return f.byEmail, f.emailErr
}

func (f *fakeUserRepository) FindUserByID(context.Context, string) (user.User, error) {
	return f.byID, f.idErr
}

type fakeSessionService struct {
	input  session.CreateSessionInput
	result session.CreateSessionResult
	err    error
	calls  int
}

func (f *fakeSessionService) CreateSession(_ context.Context, input session.CreateSessionInput) (session.CreateSessionResult, error) {
	f.calls++
	f.input = input
	return f.result, f.err
}

func (f *fakeSessionService) AuthenticateSession(context.Context, string) (session.Session, error) {
	return session.Session{}, nil
}

func (f *fakeSessionService) RevokeSession(context.Context, string) error { return nil }

func (f *fakeSessionService) RevokeAllUserSessions(context.Context, string) (int, error) {
	return 0, nil
}

type fakeRateLimiter struct {
	err  error
	keys []string
}

func (f *fakeRateLimiter) Allow(_ context.Context, key string) error {
	f.keys = append(f.keys, key)
	return f.err
}

func newTestService(t *testing.T, users user.Repository, sessions session.Service, ids IDGenerator, limiter RateLimiter) Service {
	t.Helper()
	service, err := NewService(users, sessions, fixedClock{now: time.Date(2026, 9, 21, 20, 0, 0, 0, time.UTC)}, ids, limiter)
	if err != nil {
		t.Fatalf("NewService returned an unexpected error: %v", err)
	}
	return service
}

func TestRegister_HashesPasswordAndDoesNotCreateSession(t *testing.T) {
	users := &fakeUserRepository{}
	sessions := &fakeSessionService{}
	service := newTestService(t, users, sessions, &fixedIDs{values: []string{"user-id"}}, &fakeRateLimiter{})

	created, err := service.Register(context.Background(), RegisterInput{Email: " Learner@Example.COM ", Password: "correct horse battery"})
	if err != nil {
		t.Fatalf("Register returned an unexpected error: %v", err)
	}
	if sessions.calls != 0 {
		t.Fatal("Register created a session")
	}
	if created.ID != "user-id" || created.Email != "learner@example.com" || created.Role != "LEARNER" || !created.IsActive {
		t.Fatalf("created user = %#v", created)
	}
	if users.created.PasswordHash == "" || users.created.PasswordHash == "correct horse battery" {
		t.Fatal("Register did not persist a password hash")
	}
	if !auth.VerifyPassword("correct horse battery", users.created.PasswordHash) {
		t.Fatal("persisted password hash does not verify")
	}
	if users.created.PasswordHash == created.PasswordHash && created.PasswordHash == "" {
		t.Fatal("public user unexpectedly contains no internal password state check")
	}
}

func TestRegister_RejectsPasswordBoundaries(t *testing.T) {
	for _, password := range []string{"short", "", "12345678901", strings.Repeat("x", 129)} {
		t.Run(password, func(t *testing.T) {
			service := newTestService(t, &fakeUserRepository{}, &fakeSessionService{}, &fixedIDs{values: []string{"user-id"}}, &fakeRateLimiter{})
			if _, err := service.Register(context.Background(), RegisterInput{Email: "user@example.com", Password: password}); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("Register error = %v; want ErrInvalidInput", err)
			}
		})
	}
}

func TestRegister_AcceptsPasswordLengthBoundaries(t *testing.T) {
	for _, length := range []int{12, 128} {
		t.Run(fmt.Sprintf("length-%d", length), func(t *testing.T) {
			service := newTestService(t, &fakeUserRepository{}, &fakeSessionService{}, &fixedIDs{values: []string{"user-id"}}, &fakeRateLimiter{})
			if _, err := service.Register(context.Background(), RegisterInput{Email: "user@example.com", Password: strings.Repeat("x", length)}); err != nil {
				t.Fatalf("Register returned an unexpected error: %v", err)
			}
		})
	}
}

func TestLogin_CreatesFreshTwentyFourHourSession(t *testing.T) {
	now := time.Date(2026, 9, 21, 20, 0, 0, 0, time.UTC)
	passwordHash, err := auth.HashPassword("correct horse battery")
	if err != nil {
		t.Fatalf("HashPassword returned an unexpected error: %v", err)
	}
	users := &fakeUserRepository{byEmail: user.User{ID: "user-id", Email: "user@example.com", PasswordHash: passwordHash, Role: "LEARNER", IsActive: true}}
	sessions := &fakeSessionService{result: session.CreateSessionResult{RawToken: "raw-token"}}
	service, err := NewService(users, sessions, fixedClock{now: now}, &fixedIDs{values: []string{"session-id"}}, &fakeRateLimiter{})
	if err != nil {
		t.Fatalf("NewService returned an unexpected error: %v", err)
	}

	result, err := service.Login(context.Background(), LoginInput{Email: "USER@example.com", Password: "correct horse battery"})
	if err != nil {
		t.Fatalf("Login returned an unexpected error: %v", err)
	}
	if sessions.calls != 1 || sessions.input.SessionID != "session-id" || sessions.input.UserID != "user-id" || !sessions.input.ExpiresAt.Equal(now.Add(24*time.Hour)) {
		t.Fatalf("session input = %#v", sessions.input)
	}
	if result.Session.RawToken != "raw-token" || result.User.ID != "user-id" {
		t.Fatalf("login result = %#v", result)
	}
}

func TestLogin_HidesInvalidAccountStates(t *testing.T) {
	passwordHash, err := auth.HashPassword("correct horse battery")
	if err != nil {
		t.Fatalf("HashPassword returned an unexpected error: %v", err)
	}
	for _, test := range []struct {
		name  string
		value user.User
		err   error
	}{
		{name: "unknown", err: user.ErrUserNotFound},
		{name: "inactive", value: user.User{PasswordHash: passwordHash}, err: nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.name == "inactive" {
				test.value.IsActive = false
			}
			service := newTestService(t, &fakeUserRepository{byEmail: test.value, emailErr: test.err}, &fakeSessionService{}, &fixedIDs{values: []string{"session-id"}}, &fakeRateLimiter{})
			_, loginErr := service.Login(context.Background(), LoginInput{Email: "user@example.com", Password: "correct horse battery"})
			if !errors.Is(loginErr, ErrInvalidCredentials) {
				t.Fatalf("Login error = %v; want ErrInvalidCredentials", loginErr)
			}
		})
	}
}

func TestRateLimiterErrorsAreHandled(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "intentional rate limit", err: ErrRateLimited, want: ErrRateLimited},
		{name: "canceled", err: context.Canceled, want: context.Canceled},
		{name: "deadline", err: context.DeadlineExceeded, want: context.DeadlineExceeded},
		{name: "unexpected", err: errors.New("limiter unavailable"), want: nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := newTestService(t, &fakeUserRepository{}, &fakeSessionService{}, &fixedIDs{values: []string{"user-id"}}, &fakeRateLimiter{err: test.err})
			registerErr := func() error {
				_, err := service.Register(context.Background(), RegisterInput{Email: "user@example.com", Password: "correct horse battery"})
				return err
			}()
			if test.want == nil {
				if !errors.Is(registerErr, test.err) {
					t.Fatalf("Register error = %v; want underlying limiter error", registerErr)
				}
			} else if !errors.Is(registerErr, test.want) {
				t.Fatalf("Register error = %v; want %v", registerErr, test.want)
			}
			loginService := newTestService(t, &fakeUserRepository{}, &fakeSessionService{}, &fixedIDs{values: []string{"session-id"}}, &fakeRateLimiter{err: test.err})
			_, loginErr := loginService.Login(context.Background(), LoginInput{Email: "user@example.com", Password: "correct horse battery"})
			if test.want == nil {
				if !errors.Is(loginErr, test.err) {
					t.Fatalf("Login error = %v; want underlying limiter error", loginErr)
				}
			} else if !errors.Is(loginErr, test.want) {
				t.Fatalf("Login error = %v; want %v", loginErr, test.want)
			}
		})
	}
}

func TestNewService_RequiresRateLimiter(t *testing.T) {
	_, err := NewService(&fakeUserRepository{}, &fakeSessionService{}, fixedClock{}, &fixedIDs{values: []string{"id"}}, nil)
	if !errors.Is(err, ErrInvalidService) {
		t.Fatalf("NewService error = %v; want ErrInvalidService", err)
	}
}

func TestIdentityOperationsUseDistinctNormalizedRateLimitKeys(t *testing.T) {
	limiter := &fakeRateLimiter{}
	passwordHash, err := auth.HashPassword("correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	service := newTestService(t, &fakeUserRepository{byEmail: user.User{ID: "user-id", Email: "user@example.com", PasswordHash: passwordHash, IsActive: true}}, &fakeSessionService{}, &fixedIDs{values: []string{"user-id", "session-id"}}, limiter)
	if _, err := service.Register(context.Background(), RegisterInput{Email: " USER@example.com ", Password: "correct horse battery"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Login(context.Background(), LoginInput{Email: " USER@example.com ", Password: "correct horse battery"}); err != nil {
		t.Fatal(err)
	}
	if len(limiter.keys) != 2 || limiter.keys[0] != "register:user@example.com" || limiter.keys[1] != "login:user@example.com" {
		t.Fatalf("limiter keys = %#v", limiter.keys)
	}
}
