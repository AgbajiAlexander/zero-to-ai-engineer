package learner

import (
	"context"
	"errors"
	"testing"
)

type fakeRepository struct {
	created     Learner
	createErr   error
	found       Learner
	findErr     error
	createCalls int
	findCalls   int
}

func (f *fakeRepository) Create(_ context.Context, value Learner) error {
	f.createCalls++
	f.created = value
	return f.createErr
}

func (f *fakeRepository) FindByUserID(_ context.Context, userID string) (Learner, error) {
	f.findCalls++
	if f.findErr != nil {
		return Learner{}, f.findErr
	}
	if f.found.UserID != userID {
		return Learner{}, ErrLearnerNotFound
	}
	return f.found, nil
}

func TestOnboardNormalisesDisplayName(t *testing.T) {
	repo := &fakeRepository{}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	got, err := service.Onboard(context.Background(), OnboardInput{
		UserID:      "user-123",
		DisplayName: "  Alice   Engineer  ",
	})
	if err != nil {
		t.Fatalf("Onboard() error = %v", err)
	}

	if got.DisplayName != "Alice Engineer" {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, "Alice Engineer")
	}
	if repo.createCalls != 1 {
		t.Errorf("Create calls = %d, want 1", repo.createCalls)
	}
	if repo.created.UserID != "user-123" {
		t.Errorf("created UserID = %q, want %q", repo.created.UserID, "user-123")
	}
}

func TestOnboardRejectsInvalidInput(t *testing.T) {
	repo := &fakeRepository{}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	tests := []struct {
		name  string
		input OnboardInput
	}{
		{
			name: "missing user ID",
			input: OnboardInput{
				DisplayName: "Alice",
			},
		},
		{
			name: "blank display name",
			input: OnboardInput{
				UserID:      "user-123",
				DisplayName: "   ",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Onboard(context.Background(), tt.input)
			if !errors.Is(err, ErrInvalidInput) {
				t.Errorf("Onboard() error = %v, want ErrInvalidInput", err)
			}
		})
	}

	if repo.createCalls != 0 {
		t.Errorf("Create calls = %d, want 0", repo.createCalls)
	}
}

func TestOnboardReturnsRepositoryError(t *testing.T) {
	repoErr := errors.New("database unavailable")
	repo := &fakeRepository{createErr: repoErr}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, err = service.Onboard(context.Background(), OnboardInput{
		UserID:      "user-123",
		DisplayName: "Alice",
	})
	if !errors.Is(err, repoErr) {
		t.Errorf("Onboard() error = %v, want wrapped repository error", err)
	}
}

func TestGetByUserIDReturnsProfile(t *testing.T) {
	repo := &fakeRepository{
		found: Learner{
			UserID:      "user-123",
			DisplayName: "Alice",
		},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	got, err := service.GetByUserID(context.Background(), "user-123")
	if err != nil {
		t.Fatalf("GetByUserID() error = %v", err)
	}
	if got.DisplayName != "Alice" {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, "Alice")
	}
	if repo.findCalls != 1 {
		t.Errorf("FindByUserID calls = %d, want 1", repo.findCalls)
	}
}

func TestGetByUserIDReturnsNotFound(t *testing.T) {
	repo := &fakeRepository{}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, err = service.GetByUserID(context.Background(), "user-123")
	if !errors.Is(err, ErrLearnerNotFound) {
		t.Errorf("GetByUserID() error = %v, want ErrLearnerNotFound", err)
	}
}

func TestNewServiceRejectsNilRepository(t *testing.T) {
	_, err := NewService(nil)
	if !errors.Is(err, ErrInvalidService) {
		t.Errorf("NewService() error = %v, want ErrInvalidService", err)
	}
}
