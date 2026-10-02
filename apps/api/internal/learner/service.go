package learner

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrInvalidInput   = errors.New("learner: invalid input")
	ErrInvalidService = errors.New("learner: invalid service")
)

// Service defines the learner profile business operations.
type Service interface {
	Onboard(ctx context.Context, input OnboardInput) (Learner, error)
	GetByUserID(ctx context.Context, userID string) (Learner, error)
}

// OnboardInput contains the data required to create a learner profile.
// UserID must come from the authenticated session, not the request body.
type OnboardInput struct {
	UserID      string
	DisplayName string
}

type service struct {
	repository Repository
}

// NewService creates a learner service with its required repository.
func NewService(repository Repository) (Service, error) {
	if repository == nil {
		return nil, ErrInvalidService
	}

	return &service{
		repository: repository,
	}, nil
}

// Onboard validates and creates a learner profile.
func (s *service) Onboard(
	ctx context.Context,
	input OnboardInput,
) (Learner, error) {
	if s == nil || s.repository == nil {
		return Learner{}, ErrInvalidService
	}

	if input.UserID == "" {
		return Learner{}, ErrInvalidInput
	}

	displayName, err := NormalizeDisplayName(input.DisplayName)
	if err != nil {
		return Learner{}, fmt.Errorf("%w: display name", ErrInvalidInput)
	}

	value := Learner{
		UserID:      input.UserID,
		DisplayName: displayName,
	}

	if err := s.repository.Create(ctx, value); err != nil {
		return Learner{}, fmt.Errorf("create learner profile: %w", err)
	}

	return value, nil
}

// GetByUserID retrieves a learner profile by its authenticated user's ID.
func (s *service) GetByUserID(
	ctx context.Context,
	userID string,
) (Learner, error) {
	if s == nil || s.repository == nil {
		return Learner{}, ErrInvalidService
	}

	if userID == "" {
		return Learner{}, ErrInvalidInput
	}

	value, err := s.repository.FindByUserID(ctx, userID)
	if err != nil {
		return Learner{}, fmt.Errorf("find learner profile: %w", err)
	}

	return value, nil
}
