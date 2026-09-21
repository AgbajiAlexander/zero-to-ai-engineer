package session

import (
	"reflect"
	"testing"
	"time"
)

func TestSession_ContainsPersistedFieldsWithoutRawToken(t *testing.T) {
	typ := reflect.TypeOf(Session{})
	if _, ok := typ.FieldByName("RawToken"); ok {
		t.Fatal("Session must not contain a raw token")
	}
	if _, ok := typ.FieldByName("Token"); ok {
		t.Fatal("Session must not contain a raw token")
	}

	lastSeen := time.Now()
	revoked := time.Now()
	session := Session{
		ID:         "session-id",
		UserID:     "user-id",
		TokenHash:  "token-hash",
		ExpiresAt:  time.Now().Add(time.Hour),
		CreatedAt:  time.Now(),
		LastSeenAt: &lastSeen,
		RevokedAt:  &revoked,
	}
	if session.TokenHash == "" || session.ID == "" || session.UserID == "" {
		t.Fatal("Session did not retain required persisted identifiers")
	}
}
