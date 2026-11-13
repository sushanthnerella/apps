package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/purohit-platform/backend/internal/models"
)

// ReserveIdempotencyKey attempts to reserve the idempotency key in an 'IN_PROGRESS' state with a 60-second lease.
// Returns true if successfully inserted/reserved, or false if a record with (caller_id, key) already exists.
func (s *Storage) ReserveIdempotencyKey(ctx context.Context, key, callerID, route, requestHash string) (bool, error) {
	query := `
		INSERT INTO idempotency_keys (key, caller_id, route, request_hash, status, locked_until, created_at)
		VALUES ($1, $2, $3, $4, 'IN_PROGRESS', NOW() + INTERVAL '60 seconds', NOW())
		ON CONFLICT (caller_id, key) DO NOTHING
		RETURNING key
	`
	var reservedKey string
	err := s.DB.QueryRowContext(ctx, query, key, callerID, route, requestHash).Scan(&reservedKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("idempotency reserve failed: %w", err)
	}
	return true, nil
}

// GetIdempotencyKey retrieves an idempotency record for a caller and key.
func (s *Storage) GetIdempotencyKey(ctx context.Context, callerID, key string) (*models.IdempotencyRecord, error) {
	query := `
		SELECT key, caller_id, route, request_hash, status, locked_until, COALESCE(response_status, 0), COALESCE(response_body, ''), created_at
		FROM idempotency_keys
		WHERE caller_id = $1 AND key = $2
	`
	var rec models.IdempotencyRecord
	var lockedUntil sql.NullTime
	err := s.DB.QueryRowContext(ctx, query, callerID, key).Scan(
		&rec.Key, &rec.CallerID, &rec.Route, &rec.RequestHash,
		&rec.Status, &lockedUntil, &rec.ResponseStatus, &rec.ResponseBody, &rec.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("idempotency query failed: %w", err)
	}
	if lockedUntil.Valid {
		rec.LockedUntil = &lockedUntil.Time
	}
	return &rec, nil
}

// UpdateIdempotencySuccess marks an in-progress idempotency key as COMPLETED with the captured status and response body.
func (s *Storage) UpdateIdempotencySuccess(ctx context.Context, key, callerID string, status int, body string) error {
	query := `
		UPDATE idempotency_keys
		SET status = 'COMPLETED', response_status = $3, response_body = $4, locked_until = NULL
		WHERE caller_id = $1 AND key = $2
	`
	_, err := s.DB.ExecContext(ctx, query, callerID, key, status, body)
	if err != nil {
		return fmt.Errorf("failed to complete idempotency key: %w", err)
	}
	return nil
}
