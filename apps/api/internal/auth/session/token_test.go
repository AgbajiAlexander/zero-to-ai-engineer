package session

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestGenerateToken_Succeeds(t *testing.T) {
	token, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken returned an unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("GenerateToken returned an empty token")
	}
}

func TestGenerateToken_HasExpectedEntropyAndEncoding(t *testing.T) {
	token, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken returned an unexpected error: %v", err)
	}

	if len(token) != base64.RawURLEncoding.EncodedLen(tokenByteLength) {
		t.Fatalf("token length = %d; want %d", len(token), base64.RawURLEncoding.EncodedLen(tokenByteLength))
	}
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		t.Fatalf("token is not valid raw URL-safe base64: %v", err)
	}
	if len(decoded) != tokenByteLength {
		t.Fatalf("decoded token length = %d; want %d", len(decoded), tokenByteLength)
	}
}

func TestGenerateToken_ProducesDifferentTokens(t *testing.T) {
	first, err := GenerateToken()
	if err != nil {
		t.Fatalf("first GenerateToken returned an unexpected error: %v", err)
	}
	second, err := GenerateToken()
	if err != nil {
		t.Fatalf("second GenerateToken returned an unexpected error: %v", err)
	}
	if first == second {
		t.Fatal("GenerateToken produced identical tokens")
	}
}

func TestHashToken_IsDeterministic(t *testing.T) {
	token := "raw-session-token"
	first, err := HashToken(token)
	if err != nil {
		t.Fatalf("first HashToken returned an unexpected error: %v", err)
	}
	second, err := HashToken(token)
	if err != nil {
		t.Fatalf("second HashToken returned an unexpected error: %v", err)
	}
	if first != second {
		t.Fatal("HashToken produced different hashes for the same token")
	}
	if first == token {
		t.Fatal("HashToken returned the plaintext token")
	}
	if len(first) != 64 {
		t.Fatalf("hash length = %d; want 64 hex characters", len(first))
	}
	if strings.ToLower(first) != first {
		t.Fatal("HashToken returned a non-lowercase hash")
	}
}

func TestHashToken_DifferentTokensProduceDifferentHashes(t *testing.T) {
	first, err := HashToken("first-raw-session-token")
	if err != nil {
		t.Fatalf("first HashToken returned an unexpected error: %v", err)
	}
	second, err := HashToken("second-raw-session-token")
	if err != nil {
		t.Fatalf("second HashToken returned an unexpected error: %v", err)
	}
	if first == second {
		t.Fatal("HashToken produced the same hash for different tokens")
	}
}

func TestHashToken_RejectsEmptyToken(t *testing.T) {
	hash, err := HashToken("")
	if err != ErrEmptyToken {
		t.Fatalf("HashToken error = %v; want %v", err, ErrEmptyToken)
	}
	if hash != "" {
		t.Fatal("HashToken returned a hash for an empty token")
	}
}
