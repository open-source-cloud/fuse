package idempotency

import (
	"context"
	"sync"
	"time"
)

type idempotencyEntry struct {
	workflowID string
	expiresAt  time.Time
}

// MemoryStore is an in-memory implementation of Store with TTL-based expiration
type MemoryStore struct {
	mu      sync.RWMutex
	entries map[string]idempotencyEntry
	cancel  context.CancelFunc
}

// NewMemoryStore creates a new in-memory idempotency store.
// The cleanup goroutine runs until the provided context is cancelled.
func NewMemoryStore(ctx context.Context) *MemoryStore {
	ctx, cancel := context.WithCancel(ctx)
	store := &MemoryStore{
		entries: make(map[string]idempotencyEntry),
		cancel:  cancel,
	}
	go store.cleanup(ctx)
	return store
}

// Check returns the workflow ID if the key has been seen and is not expired
func (s *MemoryStore) Check(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, exists := s.entries[key]
	if !exists || time.Now().After(entry.expiresAt) {
		return "", false
	}
	return entry.workflowID, true
}

// Set records an idempotency key with its associated workflow ID and TTL
func (s *MemoryStore) Set(key string, workflowID string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[key] = idempotencyEntry{
		workflowID: workflowID,
		expiresAt:  time.Now().Add(ttl),
	}
	return nil
}

// Delete removes an idempotency key
func (s *MemoryStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, key)
	return nil
}

func (s *MemoryStore) cleanup(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for key, entry := range s.entries {
				if now.After(entry.expiresAt) {
					delete(s.entries, key)
				}
			}
			s.mu.Unlock()
		}
	}
}
