package user

import (
	"context"
	"errors"
)

var (
	ErrUserNotFound      = errors.New("user: user not found")
	ErrDuplicateEmail    = errors.New("user: email already registered")
	ErrInvalidUser       = errors.New("user: invalid user")
	ErrInvalidRepository = errors.New("user: invalid repository")
)

// Repository persists users without exposing PostgreSQL-specific types.
type Repository interface {
	CreateUser(ctx context.Context, value User) error
	FindUserByEmail(ctx context.Context, email string) (User, error)
	FindUserByID(ctx context.Context, id string) (User, error)
}
