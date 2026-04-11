// Package idempotency provides idempotency key tracking for workflow triggers
package idempotency

import "time"

// Store tracks idempotency keys and their associated workflow IDs
type Store interface {
	// Check returns the workflow ID if the key has been seen, or empty string if new
	Check(key string) (workflowID string, exists bool)
	// Set records an idempotency key with its associated workflow ID and TTL
	Set(key string, workflowID string, ttl time.Duration) error
	// Delete removes an idempotency key
	Delete(key string) error
	// CheckAndSet atomically checks if a key exists and sets it if not.
	// Returns the existing workflow ID and true if the key was already present,
	// or empty string and false if it was newly set.
	// Used by cron/event triggers for cross-node deduplication.
	CheckAndSet(key string, workflowID string, ttl time.Duration) (existingID string, existed bool)
}
