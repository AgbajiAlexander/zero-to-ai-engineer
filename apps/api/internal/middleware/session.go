package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/zero-to-ai-engineer/api/internal/auth/session"
)

const SessionCookieName = "__Host-session"

type contextKey struct{}

var sessionContextKey contextKey

// SessionFromContext returns the authenticated session attached by SessionAuthentication.
func SessionFromContext(ctx context.Context) (session.Session, bool) {
	value, ok := ctx.Value(sessionContextKey).(session.Session)
	return value, ok
}

// SessionAuthentication authenticates the session cookie and stores only the domain session in context.
func SessionAuthentication(service session.Service, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(SessionCookieName)
		if err != nil || cookie.Value == "" {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		authenticated, err := service.AuthenticateSession(r.Context(), cookie.Value)
		if err != nil {
			status := http.StatusInternalServerError
			code := "internal_error"
			if errors.Is(err, session.ErrEmptyToken) ||
				errors.Is(err, session.ErrSessionNotFound) ||
				errors.Is(err, session.ErrSessionExpired) ||
				errors.Is(err, session.ErrSessionRevoked) {
				status = http.StatusUnauthorized
				code = "unauthorized"
			}
			writeJSONError(w, status, code)
			return
		}

		ctx := context.WithValue(r.Context(), sessionContextKey, authenticated)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func writeJSONError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{Error: code})
}
