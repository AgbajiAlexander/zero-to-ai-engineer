package session

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zero-to-ai-engineer/api/internal/database"
)

var ErrInvalidDatabase = errors.New("session: invalid database")

// PostgresSessionRepository persists sessions using the application's database pool.
type PostgresSessionRepository struct {
	pool *pgxpool.Pool
}

var _ Repository = (*PostgresSessionRepository)(nil)

// NewPostgresSessionRepository creates a session repository backed by the existing database pool.
func NewPostgresSessionRepository(db *database.DB) (*PostgresSessionRepository, error) {
	if db == nil || db.Pool == nil {
		return nil, ErrInvalidDatabase
	}
	return &PostgresSessionRepository{pool: db.Pool}, nil
}

// CreateSession inserts a session and its persisted token hash.
func (r *PostgresSessionRepository) CreateSession(ctx context.Context, session Session) error {
	id, userID, err := validateSessionForPersistence(session)
	if err != nil {
		return err
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO sessions (
			id, user_id, token_hash, expires_at, created_at, last_seen_at, revoked_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, id, userID, session.TokenHash, session.ExpiresAt, session.CreatedAt,
		nullableTimestamp(session.LastSeenAt), nullableTimestamp(session.RevokedAt))
	if err != nil {
		return fmt.Errorf("session: create: %w", err)
	}
	return nil
}

// FindSessionByTokenHash retrieves a session without applying authentication policy.
func (r *PostgresSessionRepository) FindSessionByTokenHash(ctx context.Context, tokenHash string) (Session, error) {
	if tokenHash == "" {
		return Session{}, ErrInvalidTokenHash
	}

	var (
		id         pgtype.UUID
		userID     pgtype.UUID
		expiresAt  time.Time
		createdAt  time.Time
		lastSeenAt pgtype.Timestamptz
		revokedAt  pgtype.Timestamptz
		result     Session
	)

	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, created_at, last_seen_at, revoked_at
		FROM sessions
		WHERE token_hash = $1
	`, tokenHash).Scan(
		&id,
		&userID,
		&result.TokenHash,
		&expiresAt,
		&createdAt,
		&lastSeenAt,
		&revokedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrSessionNotFound
	}
	if err != nil {
		return Session{}, fmt.Errorf("session: find by token hash: %w", err)
	}

	if !id.Valid || !userID.Valid {
		return Session{}, fmt.Errorf("session: find by token hash: %w", ErrInvalidSession)
	}
	result.ID = id.String()
	result.UserID = userID.String()
	result.ExpiresAt = expiresAt
	result.CreatedAt = createdAt
	result.LastSeenAt = nullableTimestampValue(lastSeenAt)
	result.RevokedAt = nullableTimestampValue(revokedAt)
	return result, nil
}

// RevokeSession marks one active session revoked without changing an existing timestamp.
func (r *PostgresSessionRepository) RevokeSession(ctx context.Context, sessionID string, revokedAt time.Time) error {
	id, err := parseUUID(sessionID, "session ID")
	if err != nil {
		return err
	}

	var status string
	err = r.pool.QueryRow(ctx, `
		WITH updated AS (
			UPDATE sessions
			SET revoked_at = $2
			WHERE id = $1 AND revoked_at IS NULL
			RETURNING id
		)
		SELECT CASE
			WHEN EXISTS (SELECT 1 FROM updated) THEN 'revoked'
			WHEN EXISTS (SELECT 1 FROM sessions WHERE id = $1) THEN 'already_revoked'
			ELSE 'not_found'
		END
	`, id, revokedAt).Scan(&status)
	if err != nil {
		return fmt.Errorf("session: revoke: %w", err)
	}

	switch status {
	case "revoked":
		return nil
	case "already_revoked":
		return ErrSessionAlreadyRevoked
	case "not_found":
		return ErrSessionNotFound
	default:
		return fmt.Errorf("session: revoke: unexpected repository status %q", status)
	}
}

// RevokeAllUserSessions marks every currently active session for a user revoked.
func (r *PostgresSessionRepository) RevokeAllUserSessions(ctx context.Context, userID string, revokedAt time.Time) (int, error) {
	id, err := parseUUID(userID, "user ID")
	if err != nil {
		return 0, err
	}

	result, err := r.pool.Exec(ctx, `
		UPDATE sessions
		SET revoked_at = $2
		WHERE user_id = $1 AND revoked_at IS NULL
	`, id, revokedAt)
	if err != nil {
		return 0, fmt.Errorf("session: revoke all for user: %w", err)
	}
	return int(result.RowsAffected()), nil
}

func validateSessionForPersistence(session Session) (pgtype.UUID, pgtype.UUID, error) {
	if session.TokenHash == "" || session.CreatedAt.IsZero() || session.ExpiresAt.IsZero() {
		return pgtype.UUID{}, pgtype.UUID{}, ErrInvalidSession
	}
	id, err := parseUUID(session.ID, "session ID")
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, err
	}
	userID, err := parseUUID(session.UserID, "user ID")
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, err
	}
	return id, userID, nil
}

func parseUUID(value, field string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(value); err != nil {
		return pgtype.UUID{}, fmt.Errorf("%w: invalid %s", ErrInvalidSession, field)
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
