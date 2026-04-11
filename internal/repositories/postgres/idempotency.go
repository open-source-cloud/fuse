package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/open-source-cloud/fuse/internal/idempotency"
)

// IdempotencyStore implements idempotency.Store backed by PostgreSQL.
// Provides cross-node deduplication for HA deployments.
type IdempotencyStore struct {
	pool *pgxpool.Pool
}

// NewIdempotencyStore creates a new PostgreSQL-backed idempotency store.
func NewIdempotencyStore(pool *pgxpool.Pool) idempotency.Store {
	return &IdempotencyStore{pool: pool}
}

// Check returns the workflow ID if the key has been seen and is not expired.
func (s *IdempotencyStore) Check(key string) (string, bool) {
	ctx := context.Background()
	var workflowID string
	err := s.pool.QueryRow(ctx, `
		SELECT workflow_id FROM idempotency_keys
		WHERE idempotency_key = $1 AND expires_at > NOW()
	`, key).Scan(&workflowID)
	if err != nil {
		return "", false
	}
	return workflowID, true
}

// Set records an idempotency key with its associated workflow ID and TTL.
// Uses INSERT ON CONFLICT to handle races between nodes.
func (s *IdempotencyStore) Set(key string, workflowID string, ttl time.Duration) error {
	ctx := context.Background()
	_, err := s.pool.Exec(ctx, `
		INSERT INTO idempotency_keys (idempotency_key, workflow_id, expires_at)
		VALUES ($1, $2, NOW() + $3::INTERVAL)
		ON CONFLICT (idempotency_key) DO UPDATE
		SET workflow_id = EXCLUDED.workflow_id,
		    expires_at = EXCLUDED.expires_at
	`, key, workflowID, fmt.Sprintf("%d seconds", int(ttl.Seconds())))
	if err != nil {
		return fmt.Errorf("postgres/idempotency: set: %w", err)
	}
	return nil
}

// Delete removes an idempotency key.
func (s *IdempotencyStore) Delete(key string) error {
	ctx := context.Background()
	_, err := s.pool.Exec(ctx, `
		DELETE FROM idempotency_keys WHERE idempotency_key = $1
	`, key)
	if err != nil {
		return fmt.Errorf("postgres/idempotency: delete: %w", err)
	}
	return nil
}

// CheckAndSet atomically checks if a key exists and sets it if not.
// Returns the existing workflow ID and true if the key was already present,
// or empty string and false if it was newly set.
// This is used by cron/event triggers for cross-node deduplication.
func (s *IdempotencyStore) CheckAndSet(key string, workflowID string, ttl time.Duration) (string, bool) {
	ctx := context.Background()

	// Attempt insert; if key exists and not expired, return existing workflow ID
	var existingWfID string
	err := s.pool.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO idempotency_keys (idempotency_key, workflow_id, expires_at)
			VALUES ($1, $2, NOW() + $3::INTERVAL)
			ON CONFLICT (idempotency_key) DO NOTHING
			RETURNING workflow_id
		)
		SELECT COALESCE(
			(SELECT workflow_id FROM inserted),
			(SELECT workflow_id FROM idempotency_keys WHERE idempotency_key = $1 AND expires_at > NOW())
		)
	`, key, workflowID, fmt.Sprintf("%d seconds", int(ttl.Seconds()))).Scan(&existingWfID)

	if err != nil {
		if err == pgx.ErrNoRows {
			// Key expired between conflict check and select — treat as new
			return "", false
		}
		return "", false
	}

	if existingWfID == workflowID {
		// We just inserted it — it's new
		return "", false
	}
	// Key was already present with a different workflow ID
	return existingWfID, true
}
