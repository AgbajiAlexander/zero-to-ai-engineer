package learner

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository stores learner profiles in PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a repository using an existing connection pool.
func NewPostgresRepository(pool *pgxpool.Pool) (*PostgresRepository, error) {
	if pool == nil {
		return nil, ErrInvalidRepository
	}

	return &PostgresRepository{
		pool: pool,
	}, nil
}

// Create inserts a learner profile into the database.
func (r *PostgresRepository) Create(ctx context.Context, value Learner) error {
	if r == nil || r.pool == nil {
		return ErrInvalidRepository
	}

	const query = `
		INSERT INTO learners (user_id, display_name)
		VALUES ($1, $2)
	`

	_, err := r.pool.Exec(ctx, query, value.UserID, value.DisplayName)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505":
				return ErrLearnerExists
			case "23503":
				return fmt.Errorf("learner user reference: %w", err)
			}
		}

		return fmt.Errorf("insert learner profile: %w", err)
	}

	return nil
}

// FindByUserID retrieves a learner profile using its user ID.
func (r *PostgresRepository) FindByUserID(
	ctx context.Context,
	userID string,
) (Learner, error) {
	if r == nil || r.pool == nil {
		return Learner{}, ErrInvalidRepository
	}

	const query = `
		SELECT user_id, display_name
		FROM learners
		WHERE user_id = $1
	`

	var value Learner
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&value.UserID,
		&value.DisplayName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Learner{}, ErrLearnerNotFound
		}

		return Learner{}, fmt.Errorf("select learner profile: %w", err)
	}

	return value, nil
}
