package learner

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ErrInvalidDisplayName indicates that a display name does not
// satisfy the learner profile validation rules.
var ErrInvalidDisplayName = errors.New("learner: invalid display name")

// Learner represents a learner's profile.
//
// UserID comes from the authenticated account. It must never be
// accepted as an ownership identifier from an HTTP request body.
type Learner struct {
	UserID      string
	DisplayName string
}

// NormalizeDisplayName validates and normalises a display name.
//
// Rules:
//   - The name must not be empty after trimming whitespace.
//   - Unicode control characters are rejected.
//   - Consecutive whitespace is collapsed into a single space.
//   - The name must contain no more than 80 Unicode code points.
//   - Legitimate Unicode characters are preserved.
func NormalizeDisplayName(input string) (string, error) {
	if !utf8.ValidString(input) {
		return "", ErrInvalidDisplayName
	}

	// Reject control characters before normalising whitespace.
	for _, r := range input {
		if unicode.IsControl(r) {
			return "", ErrInvalidDisplayName
		}
	}

	normalised := strings.Join(strings.Fields(input), " ")

	if normalised == "" {
		return "", ErrInvalidDisplayName
	}

	if utf8.RuneCountInString(normalised) > 80 {
		return "", ErrInvalidDisplayName
	}

	return normalised, nil
}
