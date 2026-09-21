package httpapi

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zero-to-ai-engineer/api/internal/auth/session"
	"github.com/zero-to-ai-engineer/api/internal/middleware"
)

type fakeService struct {
	authenticated session.Session
	authError     error
	revokeID      string
	revokeError   error
}

func (f *fakeService) CreateSession(context.Context, session.CreateSessionInput) (session.CreateSessionResult, error) {
	return session.CreateSessionResult{}, nil
}

func (f *fakeService) AuthenticateSession(context.Context, string) (session.Session, error) {
	if f.authError != nil {
		return session.Session{}, f.authError
	}
	return f.authenticated, nil
}

func (f *fakeService) RevokeSession(_ context.Context, sessionID string) error {
	f.revokeID = sessionID
	return f.revokeError
}

func (f *fakeService) RevokeAllUserSessions(context.Context, string) (int, error) {
	return 0, nil
}

func newTestRouter(t *testing.T, service session.Service) http.Handler {
	t.Helper()
	ready := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})
	router, err := NewRouter(service, ready, "https://frontend.example")
	if err != nil {
		t.Fatalf("NewRouter returned an unexpected error: %v", err)
	}
	return router
}

func TestRouter_HealthAndReady(t *testing.T) {
	router := newTestRouter(t, &fakeService{})
	for _, path := range []string{"/health", "/ready"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d; want %d", recorder.Code, http.StatusOK)
			}
		})
	}
}

func TestRouter_Logout(t *testing.T) {
	tests := []struct {
		name          string
		service       *fakeService
		method        string
		origin        string
		cookie        bool
		status        int
		wantRevoke    string
		wantSetCookie bool
	}{
		{name: "valid session", service: &fakeService{authenticated: session.Session{ID: "session-id"}}, method: http.MethodPost, origin: "https://frontend.example", cookie: true, status: http.StatusNoContent, wantRevoke: "session-id", wantSetCookie: true},
		{name: "missing cookie", service: &fakeService{}, method: http.MethodPost, origin: "https://frontend.example", status: http.StatusUnauthorized},
		{name: "missing origin", service: &fakeService{}, method: http.MethodPost, status: http.StatusForbidden},
		{name: "unexpected authentication error", service: &fakeService{authError: errors.New("database failure")}, method: http.MethodPost, origin: "https://frontend.example", cookie: true, status: http.StatusInternalServerError},
		{name: "unexpected revoke error", service: &fakeService{authenticated: session.Session{ID: "session-id"}, revokeError: errors.New("database failure")}, method: http.MethodPost, origin: "https://frontend.example", cookie: true, status: http.StatusInternalServerError},
		{name: "unexpected origin", service: &fakeService{authenticated: session.Session{ID: "session-id"}}, method: http.MethodPost, origin: "https://unexpected.example", cookie: true, status: http.StatusForbidden},
		{name: "unsupported method", service: &fakeService{}, method: http.MethodGet, origin: "https://frontend.example", status: http.StatusMethodNotAllowed},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := newTestRouter(t, test.service)
			req := httptest.NewRequest(test.method, "/api/v1/auth/logout", nil)
			req.Header.Set("Origin", test.origin)
			if test.cookie {
				req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: "raw-token"})
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			if recorder.Code != test.status {
				t.Fatalf("status = %d; want %d", recorder.Code, test.status)
			}
			if test.name == "missing cookie" && recorder.Body.String() != "{\"error\":\"unauthorized\"}\n" {
				t.Fatalf("body = %q; want structured unauthorized error", recorder.Body.String())
			}
			if test.wantRevoke != "" && test.service.revokeID != test.wantRevoke {
				t.Fatalf("revoked session ID = %q; want %q", test.service.revokeID, test.wantRevoke)
			}
			if test.wantSetCookie {
				cookies := recorder.Result().Cookies()
				if len(cookies) != 1 || cookies[0].Name != middleware.SessionCookieName || cookies[0].MaxAge != -1 ||
					!cookies[0].Secure || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode || cookies[0].Path != "/" || cookies[0].Domain != "" {
					t.Fatalf("logout cookie = %#v", cookies)
				}
			}
		})
	}
}

func TestRouter_LogoutIgnoresClientSessionID(t *testing.T) {
	service := &fakeService{authenticated: session.Session{ID: "context-session-id"}}
	router := newTestRouter(t, service)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout?sessionID=client-session-id", bytes.NewBufferString(`{"sessionID":"client-session-id"}`))
	req.Header.Set("Origin", "https://frontend.example")
	req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: "raw-token"})
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent || service.revokeID != "context-session-id" {
		t.Fatalf("status=%d, revoked ID=%q", recorder.Code, service.revokeID)
	}
}

func TestRouter_Preflight(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		origin  string
		method  string
		headers string
		status  int
	}{
		{name: "valid", path: "/api/v1/auth/logout", origin: "https://frontend.example", method: http.MethodPost, headers: "Content-Type", status: http.StatusNoContent},
		{name: "invalid method", path: "/api/v1/auth/logout", origin: "https://frontend.example", method: http.MethodGet, headers: "Content-Type", status: http.StatusForbidden},
		{name: "invalid headers", path: "/api/v1/auth/logout", origin: "https://frontend.example", method: http.MethodPost, headers: "X-Secret", status: http.StatusForbidden},
		{name: "unexpected origin", path: "/api/v1/auth/logout", origin: "https://unexpected.example", method: http.MethodPost, headers: "Content-Type", status: http.StatusForbidden},
		{name: "unrelated route", path: "/health", origin: "https://frontend.example", method: http.MethodPost, headers: "Content-Type", status: http.StatusNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := newTestRouter(t, &fakeService{})
			req := httptest.NewRequest(http.MethodOptions, test.path, nil)
			req.Header.Set("Origin", test.origin)
			req.Header.Set("Access-Control-Request-Method", test.method)
			req.Header.Set("Access-Control-Request-Headers", test.headers)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			if recorder.Code != test.status {
				t.Fatalf("status = %d; want %d", recorder.Code, test.status)
			}
			if test.name == "valid" {
				if recorder.Header().Get("Access-Control-Allow-Origin") != "https://frontend.example" || recorder.Header().Get("Access-Control-Allow-Credentials") != "true" {
					t.Fatal("valid preflight did not return explicit credentialed CORS headers")
				}
				if recorder.Header().Get("Access-Control-Allow-Origin") == "*" {
					t.Fatal("preflight returned wildcard origin with credentials")
				}
			}
		})
	}
}

func TestRouter_MethodNotAllowedIncludesAllowHeader(t *testing.T) {
	router := newTestRouter(t, &fakeService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/logout", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusMethodNotAllowed || recorder.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("status=%d, Allow=%q", recorder.Code, recorder.Header().Get("Allow"))
	}
}

func TestRouter_LogoutIsIdempotentForRevocationState(t *testing.T) {
	service := &fakeService{authenticated: session.Session{ID: "session-id"}, revokeError: session.ErrSessionAlreadyRevoked}
	router := newTestRouter(t, service)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set("Origin", "https://frontend.example")
	req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: "raw-token"})
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d; want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestRouter_RejectsUnknownRouteWithStructuredError(t *testing.T) {
	router := newTestRouter(t, &fakeService{})
	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNotFound || recorder.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("status/content type = %d/%q", recorder.Code, recorder.Header().Get("Content-Type"))
	}
}
