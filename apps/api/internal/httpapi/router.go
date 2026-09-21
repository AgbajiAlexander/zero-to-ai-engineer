package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/zero-to-ai-engineer/api/internal/auth/identity"
	"github.com/zero-to-ai-engineer/api/internal/auth/session"
	"github.com/zero-to-ai-engineer/api/internal/auth/user"
	"github.com/zero-to-ai-engineer/api/internal/middleware"
)

type Router struct {
	sessionService  session.Service
	identityService identity.Service
	readyHandler    http.Handler
	allowedOrigin   string
	logout          http.Handler
	register        http.Handler
	login           http.Handler
	me              http.Handler
}

// NewRouter creates the application HTTP boundary without exposing database infrastructure.
func NewRouter(sessionService session.Service, identityService identity.Service, readyHandler http.Handler, allowedOrigin string) (http.Handler, error) {
	if sessionService == nil || identityService == nil || readyHandler == nil {
		return nil, errors.New("httpapi: session service and ready handler are required")
	}
	router := &Router{
		sessionService:  sessionService,
		identityService: identityService,
		readyHandler:    readyHandler,
		allowedOrigin:   allowedOrigin,
	}
	router.logout = router.buildLogoutHandler()
	router.register = router.buildRegisterHandler()
	router.login = router.buildLoginHandler()
	router.me = router.buildMeHandler()
	return router, nil
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodOptions {
		r.handleOptions(w, req)
		return
	}

	r.setCORSHeaders(w, req)
	switch {
	case req.URL.Path == "/health":
		if req.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	case req.URL.Path == "/ready":
		if req.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
			return
		}
		r.readyHandler.ServeHTTP(w, req)
	case req.URL.Path == "/api/v1/auth/logout":
		if req.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
			return
		}
		r.logout.ServeHTTP(w, req)
	case req.URL.Path == "/api/v1/auth/register":
		if req.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
			return
		}
		r.register.ServeHTTP(w, req)
	case req.URL.Path == "/api/v1/auth/login":
		if req.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
			return
		}
		r.login.ServeHTTP(w, req)
	case req.URL.Path == "/api/v1/me":
		if req.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
			return
		}
		r.me.ServeHTTP(w, req)
	default:
		writeJSONError(w, http.StatusNotFound, "not_found")
	}
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type publicUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func toPublicUser(value user.User) publicUser {
	return publicUser{ID: value.ID, Email: value.Email, Role: value.Role}
}

func (r *Router) buildRegisterHandler() http.Handler {
	return validateOrigin(r.allowedOrigin, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var input authRequest
		if err := decodeJSON(req, &input); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_request")
			return
		}
		created, err := r.identityService.Register(req.Context(), identity.RegisterInput{Email: input.Email, Password: input.Password})
		if err != nil {
			writeIdentityError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]publicUser{"user": toPublicUser(created)})
	}))
}

func (r *Router) buildLoginHandler() http.Handler {
	return validateOrigin(r.allowedOrigin, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var input authRequest
		if err := decodeJSON(req, &input); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_request")
			return
		}
		result, err := r.identityService.Login(req.Context(), identity.LoginInput{Email: input.Email, Password: input.Password})
		if err != nil {
			writeIdentityError(w, err)
			return
		}
		http.SetCookie(w, newSessionCookie(result.Session.RawToken, result.Session.Session.ExpiresAt))
		writeJSON(w, http.StatusOK, map[string]publicUser{"user": toPublicUser(result.User)})
	}))
}

func (r *Router) buildMeHandler() http.Handler {
	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		authenticated, ok := middleware.SessionFromContext(req.Context())
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		value, err := r.identityService.GetUser(req.Context(), authenticated.UserID)
		if err != nil {
			if errors.Is(err, identity.ErrInactiveAccount) || errors.Is(err, user.ErrUserNotFound) {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			writeIdentityError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]publicUser{"user": toPublicUser(value)})
	})
	return middleware.SessionAuthentication(r.sessionService, handler)
}

func decodeJSON(req *http.Request, value any) error {
	if req.Body == nil {
		return errors.New("httpapi: request body is required")
	}
	decoder := json.NewDecoder(io.LimitReader(req.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("httpapi: request body must contain one JSON value")
	}
	return nil
}

func writeIdentityError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, identity.ErrInvalidInput):
		writeJSONError(w, http.StatusBadRequest, "invalid_request")
	case errors.Is(err, identity.ErrInvalidCredentials), errors.Is(err, identity.ErrInactiveAccount):
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
	case errors.Is(err, identity.ErrRateLimited):
		writeJSONError(w, http.StatusTooManyRequests, "rate_limited")
	case errors.Is(err, user.ErrDuplicateEmail):
		writeJSONError(w, http.StatusConflict, "email_already_registered")
	default:
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
	}
}

func (r *Router) buildLogoutHandler() http.Handler {
	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		authenticated, ok := middleware.SessionFromContext(req.Context())
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		err := r.sessionService.RevokeSession(req.Context(), authenticated.ID)
		if err != nil && !errors.Is(err, session.ErrSessionNotFound) && !errors.Is(err, session.ErrSessionAlreadyRevoked) {
			writeJSONError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		http.SetCookie(w, clearSessionCookie())
		w.WriteHeader(http.StatusNoContent)
	})
	return validateOrigin(r.allowedOrigin, middleware.SessionAuthentication(r.sessionService, handler))
}

func validateOrigin(allowedOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		origin := req.Header.Get("Origin")
		if allowedOrigin == "" || origin != allowedOrigin {
			writeJSONError(w, http.StatusForbidden, "forbidden")
			return
		}
		next.ServeHTTP(w, req)
	})
}

func (r *Router) handleOptions(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/api/v1/auth/logout" && req.URL.Path != "/api/v1/auth/register" && req.URL.Path != "/api/v1/auth/login" {
		writeJSONError(w, http.StatusNotFound, "not_found")
		return
	}
	origin := req.Header.Get("Origin")
	requestedMethod := req.Header.Get("Access-Control-Request-Method")
	requestedHeaders := req.Header.Get("Access-Control-Request-Headers")
	if r.allowedOrigin == "" || origin != r.allowedOrigin || requestedMethod != http.MethodPost || !allowedPreflightHeaders(requestedHeaders) {
		writeJSONError(w, http.StatusForbidden, "forbidden")
		return
	}
	r.setCORSHeaders(w, req)
	w.Header().Set("Access-Control-Allow-Methods", http.MethodPost)
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(http.StatusNoContent)
}

func allowedPreflightHeaders(value string) bool {
	if value == "" {
		return true
	}
	for _, header := range strings.Split(value, ",") {
		if strings.TrimSpace(strings.ToLower(header)) != "content-type" {
			return false
		}
	}
	return true
}

func (r *Router) setCORSHeaders(w http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Origin") == r.allowedOrigin && r.allowedOrigin != "" {
		w.Header().Set("Access-Control-Allow-Origin", r.allowedOrigin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Add("Vary", "Origin")
	}
}

func clearSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func newSessionCookie(rawToken string, expiresAt time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    rawToken,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeJSONError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, struct {
		Error string `json:"error"`
	}{Error: code})
}
