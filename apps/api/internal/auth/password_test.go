package auth

import (
	"strings"
	"testing"
)

func TestHashPassword_HashesValidPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword returned an unexpected error: %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword returned an empty hash")
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("hash does not look like an Argon2id hash: %q", hash)
	}
}

func TestVerifyPassword_SuccessfulVerification(t *testing.T) {
	password := "correct horse battery staple"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword returned an unexpected error: %v", err)
	}
	if !VerifyPassword(password, hash) {
		t.Fatal("VerifyPassword returned false for the correct password")
	}
}

func TestVerifyPassword_FailedVerificationWithWrongPassword(t *testing.T) {
	password := "correct horse battery staple"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword returned an unexpected error: %v", err)
	}
	if VerifyPassword("wrong password", hash) {
		t.Fatal("VerifyPassword returned true for an incorrect password")
	}
}

func TestHashPassword_UsesUniqueSalts(t *testing.T) {
	password := "correct horse battery staple"

	hash1, err := HashPassword(password)
	if err != nil {
		t.Fatalf("first HashPassword returned an unexpected error: %v", err)
	}
	hash2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("second HashPassword returned an unexpected error: %v", err)
	}
	if hash1 == hash2 {
		t.Fatal("HashPassword produced identical hashes for the same password; unique salts are required")
	}
}

func TestVerifyPassword_RejectsMalformedHashes(t *testing.T) {
	if VerifyPassword("secret", "not-a-valid-argon2-hash") {
		t.Fatal("VerifyPassword accepted an invalid hash")
	}
	if VerifyPassword("secret", "") {
		t.Fatal("VerifyPassword accepted an empty hash")
	}
	if VerifyPassword("", "something") {
		t.Fatal("VerifyPassword accepted an empty password")
	}
}

func TestHashPassword_EmptyPasswordPolicy(t *testing.T) {
	if _, err := HashPassword(""); err == nil {
		t.Fatal("HashPassword should reject empty passwords")
	}
	if _, err := HashPassword(""); err != ErrEmptyPassword {
		t.Fatalf("HashPassword returned %v; want %v", err, ErrEmptyPassword)
	}
}

func TestVerifyPassword_RejectsMissingOrDuplicateArgon2Parameters(t *testing.T) {
	password := "correct horse battery staple"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword returned an unexpected error: %v", err)
	}

	missingM := strings.Replace(hash, "m=65536", "", 1)
	if VerifyPassword(password, missingM) {
		t.Fatal("VerifyPassword accepted a hash missing m")
	}

	missingT := strings.Replace(hash, "t=3", "", 1)
	if VerifyPassword(password, missingT) {
		t.Fatal("VerifyPassword accepted a hash missing t")
	}

	missingP := strings.Replace(hash, "p=2", "", 1)
	if VerifyPassword(password, missingP) {
		t.Fatal("VerifyPassword accepted a hash missing p")
	}

	duplicateM := strings.Replace(hash, "m=65536,t=3,p=2", "m=65536,m=65536,t=3,p=2", 1)
	if VerifyPassword(password, duplicateM) {
		t.Fatal("VerifyPassword accepted a hash with duplicate m")
	}

	duplicateT := strings.Replace(hash, "m=65536,t=3,p=2", "m=65536,t=3,t=3,p=2", 1)
	if VerifyPassword(password, duplicateT) {
		t.Fatal("VerifyPassword accepted a hash with duplicate t")
	}

	duplicateP := strings.Replace(hash, "m=65536,t=3,p=2", "m=65536,t=3,p=2,p=2", 1)
	if VerifyPassword(password, duplicateP) {
		t.Fatal("VerifyPassword accepted a hash with duplicate p")
	}
}

func TestVerifyPassword_AcceptsValidArgon2Parameters(t *testing.T) {
	password := "correct horse battery staple"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword returned an unexpected error: %v", err)
	}
	if !VerifyPassword(password, hash) {
		t.Fatal("VerifyPassword rejected a valid Argon2id hash")
	}
}
