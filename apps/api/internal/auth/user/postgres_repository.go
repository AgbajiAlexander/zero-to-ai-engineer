package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zero-to-ai-engineer/api/internal/database"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

var _ Repository = (*PostgresRepository)(nil)

// NewPostgresRepository creates a user repository backed by the existing database pool.
func NewPostgresRepository(db *database.DB) (Repository, error) {
	if db == nil || db.Pool == nil {
		return nil, ErrInvalidRepository
	}
	return &PostgresRepository{pool: db.Pool}, nil
}

func (r *PostgresRepository) CreateUser(ctx context.Context, value User) error {
	id, err := parseUUID(value.ID)
	if err != nil || value.Email == "" || value.PasswordHash == "" || value.Role == "" || value.CreatedAt.IsZero() || value.UpdatedAt.IsZero() {
		return ErrInvalidUser
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO users (
			id, email, password_hash, role, email_verified_at,
			is_active, created_at, updated_at, deleted_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, id, value.Email, value.PasswordHash, value.Role,
		nullableTimestamp(value.EmailVerifiedAt), value.IsActive,
		value.CreatedAt, value.UpdatedAt, nullableTimestamp(value.DeletedAt))
	if err != nil {
		return mapCreateError(err)
	}
	return nil
}

func (r *PostgresRepository) FindUserByEmail(ctx context.Context, email string) (User, error) {
	if email == "" {
		return User{}, ErrInvalidUser
	}
	return r.findUser(ctx, `
		SELECT id, email, password_hash, role, email_verified_at,
		       is_active, created_at, updated_at, deleted_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`, email)
}

func (r *PostgresRepository) FindUserByID(ctx context.Context, id string) (User, error) {
	parsedID, err := parseUUID(id)
	if err != nil {
		return User{}, ErrInvalidUser
	}
	return r.findUser(ctx, `
		SELECT id, email, password_hash, role, email_verified_at,
		       is_active, created_at, updated_at, deleted_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`, parsedID)
}

func (r *PostgresRepository) findUser(ctx context.Context, query string, args ...any) (User, error) {
	var (
		id              pgtype.UUID
		email           string
		passwordHash    string
		role            string
		emailVerifiedAt pgtype.Timestamptz
		isActive        bool
		createdAt       time.Time
		updatedAt       time.Time
		deletedAt       pgtype.Timestamptz
	)

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&id, &email, &passwordHash, &role, &emailVerifiedAt,
		&isActive, &createdAt, &updatedAt, &deletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("user: find: %w", err)
	}
	if !id.Valid {
		return User{}, ErrInvalidUser
	}

	return User{
		ID:              id.String(),
		Email:           email,
		PasswordHash:    passwordHash,
		Role:            role,
		EmailVerifiedAt: nullableTimestampValue(emailVerifiedAt),
		IsActive:        isActive,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
		DeletedAt:       nullableTimestampValue(deletedAt),
	}, nil
}

func parseUUID(value string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(value); err != nil {
		return pgtype.UUID{}, err
	}
	return id, nil
}

func nullableTimestamp(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}

func nullableTimestampValue(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

func mapCreateError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "users_email_key" {
		return ErrDuplicateEmail
	}
	return fmt.Errorf("user: create: %w", err)
}
