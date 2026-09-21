package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argon2TimeCost   = 3
	argon2MemoryCost = 64 * 1024
	argon2Threads    = 2
	argon2KeyLength  = 32
	argon2SaltLength = 16
)

var ErrEmptyPassword = errors.New("auth: password cannot be empty")

// HashPassword creates an Argon2id hash for the provided password.
// Empty passwords are rejected to keep the password boundary deterministic and safe.
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", ErrEmptyPassword
	}

	salt := make([]byte, argon2SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("auth: generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, argon2TimeCost, argon2MemoryCost, argon2Threads, argon2KeyLength)

	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, argon2MemoryCost, argon2TimeCost, argon2Threads, encodedSalt, encodedHash), nil
}

// VerifyPassword returns true when the provided password matches the encoded Argon2id hash.
// Invalid hashes and empty values are rejected safely without exposing implementation details.
func VerifyPassword(password, encodedHash string) bool {
	if password == "" || encodedHash == "" {
		return false
	}

	params, salt, expectedHash, err := parseEncodedHash(encodedHash)
	if err != nil {
		return false
	}

	actualHash := argon2.IDKey([]byte(password), salt, params.timeCost, params.memoryCost, params.threads, uint32(len(expectedHash)))
	return subtle.ConstantTimeCompare(actualHash, expectedHash) == 1
}

type argon2Params struct {
	memoryCost uint32
	timeCost   uint32
	threads    uint8
}

func parseEncodedHash(encodedHash string) (argon2Params, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return argon2Params{}, nil, nil, errors.New("auth: invalid encoded hash")
	}
	if parts[0] != "" {
		return argon2Params{}, nil, nil, errors.New("auth: invalid encoded hash prefix")
	}
	if parts[1] != "argon2id" {
		return argon2Params{}, nil, nil, errors.New("auth: unsupported hash type")
	}
	if parts[2] != fmt.Sprintf("v=%d", argon2.Version) {
		return argon2Params{}, nil, nil, errors.New("auth: unsupported argon2 version")
	}

	params := argon2Params{}
	seen := map[string]bool{}
	for _, field := range strings.Split(parts[3], ",") {
		if field == "" {
			return argon2Params{}, nil, nil, errors.New("auth: malformed hash parameters")
		}
		valueParts := strings.SplitN(field, "=", 2)
		if len(valueParts) != 2 {
			return argon2Params{}, nil, nil, errors.New("auth: malformed hash parameters")
		}
		key := valueParts[0]
		if seen[key] {
			return argon2Params{}, nil, nil, fmt.Errorf("auth: duplicate hash parameter %q", key)
		}
		seen[key] = true
		value, err := strconv.Atoi(valueParts[1])
		if err != nil {
			return argon2Params{}, nil, nil, fmt.Errorf("auth: invalid hash parameter %q: %w", key, err)
		}

		switch key {
		case "m":
			params.memoryCost = uint32(value)
		case "t":
			params.timeCost = uint32(value)
		case "p":
			params.threads = uint8(value)
		default:
			return argon2Params{}, nil, nil, fmt.Errorf("auth: unknown hash parameter %q", key)
		}
	}
	if params.memoryCost == 0 {
		return argon2Params{}, nil, nil, errors.New("auth: missing m parameter")
	}
	if params.timeCost == 0 {
		return argon2Params{}, nil, nil, errors.New("auth: missing t parameter")
	}
	if params.threads == 0 {
		return argon2Params{}, nil, nil, errors.New("auth: missing p parameter")
	}

	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil {
		return argon2Params{}, nil, nil, fmt.Errorf("auth: decode salt: %w", err)
	}
	if len(salt) == 0 {
		return argon2Params{}, nil, nil, errors.New("auth: empty salt")
	}

	expectedHash, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil {
		return argon2Params{}, nil, nil, fmt.Errorf("auth: decode hash: %w", err)
	}
	if len(expectedHash) == 0 {
		return argon2Params{}, nil, nil, errors.New("auth: empty hash")
	}

	return params, salt, expectedHash, nil
}
