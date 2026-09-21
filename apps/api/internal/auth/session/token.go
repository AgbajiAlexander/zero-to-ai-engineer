package session

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
)

const tokenByteLength = 32

var ErrEmptyToken = errors.New("session: token cannot be empty")

// GenerateToken creates a cryptographically secure raw session token.
func GenerateToken() (string, error) {
	rawToken := make([]byte, tokenByteLength)
	if _, err := rand.Read(rawToken); err != nil {
		return "", fmt.Errorf("session: generate token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(rawToken), nil
}

// HashToken returns the SHA-256 digest of a raw session token as lowercase hex.
func HashToken(rawToken string) (string, error) {
	if rawToken == "" {
		return "", ErrEmptyToken
	}

	digest := sha256.Sum256([]byte(rawToken))
	return fmt.Sprintf("%x", digest), nil
}
