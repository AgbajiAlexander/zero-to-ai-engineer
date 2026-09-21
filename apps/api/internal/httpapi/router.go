package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/zero-to-ai-engineer/api/internal/auth/session"
	"github.com/zero-to-ai-engineer/api/internal/middleware"
)

type Router struct {
	sessionService session.Service
	readyHandler   http.Handler
	allowedOrigin  string
	logout         http.Handler
}

// NewRouter creates the application HTTP boundary without exposing database infrastructure.
func NewRouter(sessionService session.Service, readyHandler http.Handler, allowedOrigin string) (http.Handler, error) {
	if sessionService == nil || readyHandler == nil {
		return nil, errors.New("httpapi: session service and ready handler are required")
	}
	router := &Router{
		sessionService: sessionService,
		readyHandler:   readyHandler,
		allowedOrigin:  allowedOrigin,
	}
	router.logout = router.buildLogoutHandler()
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
	default:
		writeJSONError(w, http.StatusNotFound, "not_found")
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
	if req.URL.Path != "/api/v1/auth/logout" {
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
