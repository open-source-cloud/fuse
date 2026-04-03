package workflow

import "sync"

// RetryTracker tracks retry attempts per execution step
type RetryTracker struct {
	mu       sync.Mutex
	attempts map[string]int // execID -> attempt count
}

// NewRetryTracker creates a new RetryTracker
func NewRetryTracker() *RetryTracker {
	return &RetryTracker{attempts: make(map[string]int)}
}

// Increment increments and returns the attempt count for a given execID
func (rt *RetryTracker) Increment(execID string) int {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.attempts[execID]++
	return rt.attempts[execID]
}

// GetAttempts returns the current attempt count for a given execID
func (rt *RetryTracker) GetAttempts(execID string) int {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.attempts[execID]
}

// Clear removes tracking for a given execID
func (rt *RetryTracker) Clear(execID string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	delete(rt.attempts, execID)
}
