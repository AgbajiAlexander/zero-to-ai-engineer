package learner

import (
	"context"
	"errors"
)

var (
	ErrLearnerNotFound   = errors.New("learner: profile not found")
	ErrLearnerExists     = errors.New("learner: profile already exists")
	ErrInvalidRepository = errors.New("learner: invalid repository")
)

// Repository defines the persistence operations for learner profiles.
//
// Implementations must use parameterised database queries and must
// respect the caller's context.
type Repository interface {
	Create(ctx context.Context, value Learner) error
	FindByUserID(ctx context.Context, userID string) (Learner, error)
}
