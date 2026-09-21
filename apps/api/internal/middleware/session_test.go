package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zero-to-ai-engineer/api/internal/auth/session"
)

type fakeSessionService struct {
	authenticated session.Session
	authError     error
	gotToken      string
}

func (f *fakeSessionService) CreateSession(context.Context, session.CreateSessionInput) (session.CreateSessionResult, error) {
	return session.CreateSessionResult{}, nil
}

func (f *fakeSessionService) AuthenticateSession(_ context.Context, rawToken string) (session.Session, error) {
	f.gotToken = rawToken
	if f.authError != nil {
		return session.Session{}, f.authError
	}
	return f.authenticated, nil
}

func (f *fakeSessionService) RevokeSession(context.Context, string) error { return nil }

func (f *fakeSessionService) RevokeAllUserSessions(context.Context, string) (int, error) {
	return 0, nil
}

func TestSessionAuthentication(t *testing.T) {
	tests := []struct {
		name       string
		cookie     string
		serviceErr error
		status     int
	}{
		{name: "missing cookie", status: http.StatusUnauthorized},
		{name: "invalid token", cookie: "raw-token", serviceErr: session.ErrSessionNotFound, status: http.StatusUnauthorized},
		{name: "expired session", cookie: "raw-token", serviceErr: session.ErrSessionExpired, status: http.StatusUnauthorized},
		{name: "revoked session", cookie: "raw-token", serviceErr: session.ErrSessionRevoked, status: http.StatusUnauthorized},
		{name: "unexpected error", cookie: "raw-token", serviceErr: errors.New("database failed"), status: http.StatusInternalServerError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &fakeSessionService{authError: test.serviceErr}
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})
			handler := SessionAuthentication(service, next)
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if test.cookie != "" {
				req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: test.cookie})
			}
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)
			if recorder.Code != test.status {
				t.Fatalf("status = %d; want %d", recorder.Code, test.status)
			}
			if recorder.Code != http.StatusNoContent && !strings.Contains(recorder.Body.String(), `"error"`) {
				t.Fatal("error response is not structured JSON")
			}
			if strings.Contains(recorder.Body.String(), test.cookie) && test.cookie != "" {
				t.Fatal("raw token leaked in response")
			}
		})
	}
}

func TestSessionAuthentication_StoresOnlySessionInContext(t *testing.T) {
	service := &fakeSessionService{authenticated: session.Session{ID: "session-id", UserID: "user-id"}}
	var got session.Session
	var ok bool
	var requestContext context.Context
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestContext = r.Context()
		got, ok = SessionFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "raw-token"})
	recorder := httptest.NewRecorder()
	SessionAuthentication(service, next).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent || !ok || got.ID != "session-id" {
		t.Fatalf("context session = %#v, ok=%v", got, ok)
	}
	if service.gotToken != "raw-token" {
		t.Fatal("middleware did not pass the cookie token to the service")
	}
	if _, present := rValue(requestContext, service.gotToken); present {
		t.Fatal("raw token was placed in request context")
	}
}

func rValue(ctx context.Context, value string) (any, bool) {
	return ctx.Value(value), ctx.Value(value) != nil
}
