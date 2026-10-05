package httpapi

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zero-to-ai-engineer/api/internal/auth/identity"
	"github.com/zero-to-ai-engineer/api/internal/auth/session"
	"github.com/zero-to-ai-engineer/api/internal/auth/user"
	"github.com/zero-to-ai-engineer/api/internal/learner"
	"github.com/zero-to-ai-engineer/api/internal/middleware"
)

type fakeService struct {
	authenticated session.Session
	authError     error
	revokeID      string
	revokeError   error
}

type fakeIdentityService struct {
	registered    user.User
	registerErr   error
	registerInput identity.RegisterInput
	loginResult   identity.LoginResult
	loginErr      error
	loginInput    identity.LoginInput
	me            user.User
	meErr         error
	meID          string
}

type fakeLearnerService struct {
	onboarded    learner.Learner
	onboardErr   error
	onboardInput learner.OnboardInput
	onboardCalls int
	profile      learner.Learner
	getErr       error
	getUserID    string
	getCalls     int
}

func (f *fakeIdentityService) Register(_ context.Context, input identity.RegisterInput) (user.User, error) {
	f.registerInput = input
	return f.registered, f.registerErr
}

func (f *fakeIdentityService) Login(_ context.Context, input identity.LoginInput) (identity.LoginResult, error) {
	f.loginInput = input
	return f.loginResult, f.loginErr
}

func (f *fakeIdentityService) GetUser(_ context.Context, id string) (user.User, error) {
	f.meID = id
	return f.me, f.meErr
}

func (f *fakeLearnerService) Onboard(_ context.Context, input learner.OnboardInput) (learner.Learner, error) {
	f.onboardCalls++
	f.onboardInput = input
	if f.onboardErr != nil {
		return learner.Learner{}, f.onboardErr
	}
	return f.onboarded, nil
}

func (f *fakeLearnerService) GetByUserID(_ context.Context, userID string) (learner.Learner, error) {
	f.getCalls++
	f.getUserID = userID
	if f.getErr != nil {
		return learner.Learner{}, f.getErr
	}
	return f.profile, nil
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
	router, err := NewRouter(service, &fakeIdentityService{}, &fakeLearnerService{}, ready, "https://frontend.example")
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

func TestRouter_SecurityHeadersCoverResponseTypes(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		setup  func(*http.Request)
		status int
	}{
		{name: "successful health response", method: http.MethodGet, path: "/health", status: http.StatusOK},
		{name: "readiness response", method: http.MethodGet, path: "/ready", status: http.StatusOK},
		{name: "unauthenticated response", method: http.MethodGet, path: "/api/v1/me", status: http.StatusUnauthorized},
		{name: "method not allowed response", method: http.MethodGet, path: "/api/v1/auth/logout", status: http.StatusMethodNotAllowed},
		{name: "not found response", method: http.MethodGet, path: "/not-found", status: http.StatusNotFound},
		{
			name:   "trusted preflight response",
			method: http.MethodOptions,
			path:   "/api/v1/auth/register",
			status: http.StatusNoContent,
			setup: func(req *http.Request) {
				req.Header.Set("Origin", "https://frontend.example")
				req.Header.Set("Access-Control-Request-Method", http.MethodPost)
				req.Header.Set("Access-Control-Request-Headers", "Content-Type")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := middleware.SecurityHeaders(newTestRouter(t, &fakeService{}))
			req := httptest.NewRequest(test.method, test.path, nil)
			if test.setup != nil {
				test.setup(req)
			}
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			if recorder.Code != test.status {
				t.Fatalf("status = %d; want %d", recorder.Code, test.status)
			}
			assertSecurityHeaders(t, recorder)

			switch test.name {
			case "unauthenticated response":
				if recorder.Header().Get("Content-Type") != "application/json" ||
					recorder.Body.String() != "{\"error\":\"unauthorized\"}\n" {
					t.Fatalf("unauthenticated response changed: content type %q, body %q",
						recorder.Header().Get("Content-Type"), recorder.Body.String())
				}
			case "method not allowed response":
				if recorder.Header().Get("Allow") != http.MethodPost {
					t.Fatalf("Allow = %q; want %q", recorder.Header().Get("Allow"), http.MethodPost)
				}
			case "trusted preflight response":
				headers := recorder.Header()
				if headers.Get("Access-Control-Allow-Origin") != "https://frontend.example" ||
					headers.Get("Access-Control-Allow-Credentials") != "true" ||
					headers.Get("Access-Control-Allow-Methods") != http.MethodPost ||
					headers.Get("Access-Control-Allow-Headers") != "Content-Type" {
					t.Fatalf("CORS headers changed: %#v", headers)
				}
			}
		})
	}
}

func TestSecurityHeadersPreserveCookiesAndExistingHeaders(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "https://frontend.example")
		w.Header().Set("Content-Type", "application/json")
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "opaque"})
		w.WriteHeader(http.StatusNoContent)
	})
	recorder := httptest.NewRecorder()

	middleware.SecurityHeaders(next).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	cookies := recorder.Result().Cookies()
	if recorder.Code != http.StatusNoContent ||
		recorder.Header().Get("Access-Control-Allow-Origin") != "https://frontend.example" ||
		recorder.Header().Get("Content-Type") != "application/json" ||
		len(cookies) != 1 || cookies[0].Name != "session" {
		t.Fatalf("existing response headers changed: status=%d headers=%#v",
			recorder.Code, recorder.Header())
	}
	assertSecurityHeaders(t, recorder)
}

func assertSecurityHeaders(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	for name, want := range map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "no-referrer",
	} {
		if got := recorder.Header().Get(name); got != want {
			t.Errorf("%s = %q; want %q", name, got, want)
		}
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

func TestRouter_Register(t *testing.T) {
	identityService := &fakeIdentityService{registered: user.User{ID: "user-id", Email: "user@example.com", Role: "LEARNER"}}
	router := newRouterWithIdentity(t, &fakeService{}, identityService)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"email":"user@example.com","password":"correct horse battery"}`))
	req.Header.Set("Origin", "https://frontend.example")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusCreated || strings.Contains(recorder.Body.String(), "password") || recorder.Header().Get("Set-Cookie") != "" {
		t.Fatalf("status/body/cookie = %d/%q/%q", recorder.Code, recorder.Body.String(), recorder.Header().Get("Set-Cookie"))
	}
	if identityService.registerInput.Email != "user@example.com" || identityService.registerInput.Password == "" {
		t.Fatal("registration input was not passed to the identity service")
	}
}

func TestRouter_LoginIssuesOnlySecureSessionCookie(t *testing.T) {
	identityService := &fakeIdentityService{
		loginResult: identity.LoginResult{
			User: user.User{ID: "user-id", Email: "user@example.com", Role: "LEARNER"},
			Session: session.CreateSessionResult{
				RawToken: "raw-token",
				Session:  session.Session{ExpiresAt: time.Now().Add(time.Hour)},
			},
		},
	}
	router := newRouterWithIdentity(t, &fakeService{}, identityService)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"user@example.com","password":"correct horse battery"}`))
	req.Header.Set("Origin", "https://frontend.example")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK || strings.Contains(recorder.Body.String(), "raw-token") || strings.Contains(recorder.Body.String(), "password") {
		t.Fatalf("status/body = %d/%q", recorder.Code, recorder.Body.String())
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %#v; want one session cookie", cookies)
	}
	cookie := cookies[0]
	if cookie.Name != middleware.SessionCookieName || cookie.Value != "raw-token" || !cookie.Secure || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" || cookie.Domain != "" {
		t.Fatalf("login cookie = %#v", cookie)
	}
}

func TestRouter_MeUsesAuthenticatedSessionUserID(t *testing.T) {
	identityService := &fakeIdentityService{me: user.User{ID: "user-id", Email: "user@example.com", Role: "LEARNER"}}
	router := newRouterWithIdentity(t, &fakeService{authenticated: session.Session{ID: "session-id", UserID: "user-id"}}, identityService)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me?userID=attacker-id", nil)
	req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: "raw-token"})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK || identityService.meID != "user-id" || strings.Contains(recorder.Body.String(), "password") {
		t.Fatalf("status/user ID/body = %d/%q/%q", recorder.Code, identityService.meID, recorder.Body.String())
	}
}

func TestRouter_PreflightIncludesIdentityRoutes(t *testing.T) {
	tests := []struct {
		path   string
		method string
	}{
		{path: "/api/v1/auth/register", method: http.MethodPost},
		{path: "/api/v1/auth/login", method: http.MethodPost},
		{path: "/api/v1/auth/logout", method: http.MethodPost},
		{path: "/api/v1/learners/onboard", method: http.MethodPost},
		{path: "/api/v1/learners/me", method: http.MethodGet},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			router := newRouterWithIdentity(t, &fakeService{}, &fakeIdentityService{})
			req := httptest.NewRequest(http.MethodOptions, test.path, nil)
			req.Header.Set("Origin", "https://frontend.example")
			req.Header.Set("Access-Control-Request-Method", test.method)
			req.Header.Set("Access-Control-Request-Headers", "Content-Type")
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			if recorder.Code != http.StatusNoContent {
				t.Fatalf("status = %d; want %d", recorder.Code, http.StatusNoContent)
			}
			if got := recorder.Header().Get("Access-Control-Allow-Methods"); got != test.method {
				t.Fatalf("Access-Control-Allow-Methods = %q; want %q", got, test.method)
			}
		})
	}
}

func TestRouter_LearnerOnboardUsesAuthenticatedUserID(t *testing.T) {
	learnerService := &fakeLearnerService{
		onboarded: learner.Learner{UserID: "authenticated-user", DisplayName: "Alice Engineer"},
	}
	router := newRouterWithLearner(
		t,
		&fakeService{authenticated: session.Session{ID: "session-id", UserID: "authenticated-user"}},
		&fakeIdentityService{},
		learnerService,
	)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/learners/onboard?user_id=attacker", strings.NewReader(`{"display_name":"Alice Engineer"}`))
	req.Header.Set("Origin", "https://frontend.example")
	req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: "raw-token"})
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d; want %d (body %q)", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	if learnerService.onboardInput.UserID != "authenticated-user" ||
		learnerService.onboardInput.DisplayName != "Alice Engineer" {
		t.Fatalf("onboard input = %#v; want authenticated user and display name", learnerService.onboardInput)
	}
	if strings.Contains(recorder.Body.String(), "attacker") || strings.Contains(recorder.Body.String(), "raw-token") {
		t.Fatalf("response leaked client-controlled identity or session token: %q", recorder.Body.String())
	}
}

func TestRouter_LearnerOnboardErrors(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "invalid input", err: learner.ErrInvalidInput, status: http.StatusBadRequest, code: "invalid_request"},
		{name: "existing profile", err: learner.ErrLearnerExists, status: http.StatusConflict, code: "learner_profile_exists"},
		{name: "unexpected service error", err: errors.New("database unavailable"), status: http.StatusInternalServerError, code: "internal_error"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			learnerService := &fakeLearnerService{onboardErr: test.err}
			router := newRouterWithLearner(
				t,
				&fakeService{authenticated: session.Session{ID: "session-id", UserID: "user-id"}},
				&fakeIdentityService{},
				learnerService,
			)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/learners/onboard", strings.NewReader(`{"display_name":"Alice"}`))
			req.Header.Set("Origin", "https://frontend.example")
			req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: "raw-token"})
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, req)
			if recorder.Code != test.status || !strings.Contains(recorder.Body.String(), `"error":"`+test.code+`"`) {
				t.Fatalf("status/body = %d/%q; want %d and error %q", recorder.Code, recorder.Body.String(), test.status, test.code)
			}
		})
	}
}

func TestRouter_LearnerOnboardRejectsClientUserID(t *testing.T) {
	learnerService := &fakeLearnerService{}
	router := newRouterWithLearner(
		t,
		&fakeService{authenticated: session.Session{ID: "session-id", UserID: "user-id"}},
		&fakeIdentityService{},
		learnerService,
	)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/learners/onboard", strings.NewReader(`{"display_name":"Alice","user_id":"attacker"}`))
	req.Header.Set("Origin", "https://frontend.example")
	req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: "raw-token"})
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusBadRequest || learnerService.onboardCalls != 0 {
		t.Fatalf("status/calls = %d/%d; want 400/0", recorder.Code, learnerService.onboardCalls)
	}
}

func TestRouter_LearnerOnboardRequiresAuthenticationAndOrigin(t *testing.T) {
	tests := []struct {
		name   string
		cookie bool
		origin string
		status int
	}{
		{name: "unauthenticated", origin: "https://frontend.example", status: http.StatusUnauthorized},
		{name: "missing origin", cookie: true, status: http.StatusForbidden},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			learnerService := &fakeLearnerService{}
			router := newRouterWithLearner(
				t,
				&fakeService{authenticated: session.Session{ID: "session-id", UserID: "user-id"}},
				&fakeIdentityService{},
				learnerService,
			)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/learners/onboard", strings.NewReader(`{"display_name":"Alice"}`))
			if test.origin != "" {
				req.Header.Set("Origin", test.origin)
			}
			if test.cookie {
				req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: "raw-token"})
			}
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, req)
			if recorder.Code != test.status || learnerService.onboardCalls != 0 {
				t.Fatalf("status/calls = %d/%d; want %d/0", recorder.Code, learnerService.onboardCalls, test.status)
			}
		})
	}
}

func TestRouter_LearnerMeUsesAuthenticatedUserID(t *testing.T) {
	learnerService := &fakeLearnerService{
		profile: learner.Learner{UserID: "authenticated-user", DisplayName: "Alice"},
	}
	router := newRouterWithLearner(
		t,
		&fakeService{authenticated: session.Session{ID: "session-id", UserID: "authenticated-user"}},
		&fakeIdentityService{},
		learnerService,
	)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/learners/me?user_id=attacker", nil)
	req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: "raw-token"})
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK || learnerService.getUserID != "authenticated-user" {
		t.Fatalf("status/user ID = %d/%q; want 200/authenticated user", recorder.Code, learnerService.getUserID)
	}
}

func TestRouter_LearnerMeNotFoundAndUnauthenticated(t *testing.T) {
	tests := []struct {
		name   string
		cookie bool
		err    error
		status int
		calls  int
	}{
		{name: "profile not found", cookie: true, err: learner.ErrLearnerNotFound, status: http.StatusNotFound, calls: 1},
		{name: "unauthenticated", status: http.StatusUnauthorized, calls: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			learnerService := &fakeLearnerService{getErr: test.err}
			router := newRouterWithLearner(
				t,
				&fakeService{authenticated: session.Session{ID: "session-id", UserID: "user-id"}},
				&fakeIdentityService{},
				learnerService,
			)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/learners/me", nil)
			if test.cookie {
				req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: "raw-token"})
			}
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, req)
			if recorder.Code != test.status || learnerService.getCalls != test.calls {
				t.Fatalf("status/calls = %d/%d; want %d/%d", recorder.Code, learnerService.getCalls, test.status, test.calls)
			}
		})
	}
}

func TestRouter_RateLimitRejectionIsStructured(t *testing.T) {
	identityService := &fakeIdentityService{loginErr: identity.ErrRateLimited}
	router := newRouterWithIdentity(t, &fakeService{}, identityService)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"user@example.com","password":"correct horse battery"}`))
	req.Header.Set("Origin", "https://frontend.example")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusTooManyRequests || recorder.Body.String() != "{\"error\":\"rate_limited\"}\n" {
		t.Fatalf("status/body = %d/%q", recorder.Code, recorder.Body.String())
	}
}

func TestRouter_LoginFailuresAreGeneric(t *testing.T) {
	for _, name := range []string{"unknown email", "wrong password"} {
		t.Run(name, func(t *testing.T) {
			router := newRouterWithIdentity(t, &fakeService{}, &fakeIdentityService{loginErr: identity.ErrInvalidCredentials})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"user@example.com","password":"correct horse battery"}`))
			req.Header.Set("Origin", "https://frontend.example")
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			if recorder.Code != http.StatusUnauthorized || recorder.Body.String() != "{\"error\":\"unauthorized\"}\n" {
				t.Fatalf("status/body = %d/%q", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestRouter_UnexpectedLimiterErrorIsInternal(t *testing.T) {
	router := newRouterWithIdentity(t, &fakeService{}, &fakeIdentityService{loginErr: errors.New("limiter unavailable")})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"user@example.com","password":"correct horse battery"}`))
	req.Header.Set("Origin", "https://frontend.example")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusInternalServerError || recorder.Body.String() != "{\"error\":\"internal_error\"}\n" {
		t.Fatalf("status/body = %d/%q", recorder.Code, recorder.Body.String())
	}
}

func TestRouter_LoginCookieUsesExactSessionExpiry(t *testing.T) {
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	expiresAt := now.Add(24 * time.Hour)
	identityService := &fakeIdentityService{loginResult: identity.LoginResult{
		User:    user.User{ID: "user-id", Email: "user@example.com", Role: "LEARNER"},
		Session: session.CreateSessionResult{RawToken: "raw-token", Session: session.Session{ExpiresAt: expiresAt}},
	}}
	router := newRouterWithIdentity(t, &fakeService{}, identityService)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"user@example.com","password":"correct horse battery"}`))
	req.Header.Set("Origin", "https://frontend.example")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].Expires.Equal(expiresAt) {
		t.Fatalf("login cookie expiry = %#v; want %v", cookies, expiresAt)
	}
}

func newRouterWithIdentity(t *testing.T, service *fakeService, identityService *fakeIdentityService) http.Handler {
	t.Helper()
	return newRouterWithLearner(t, service, identityService, &fakeLearnerService{})
}

func newRouterWithLearner(t *testing.T, service *fakeService, identityService *fakeIdentityService, learnerService *fakeLearnerService) http.Handler {
	t.Helper()
	ready := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	router, err := NewRouter(service, identityService, learnerService, ready, "https://frontend.example")
	if err != nil {
		t.Fatalf("NewRouter returned an unexpected error: %v", err)
	}
	return router
}
