package user

import "time"

// User is the persisted identity record used by the authentication boundary.
// PasswordHash is internal authentication data and must not be exposed by HTTP DTOs.
type User struct {
	ID              string
	Email           string
	PasswordHash    string
	Role            string
	EmailVerifiedAt *time.Time
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
}
