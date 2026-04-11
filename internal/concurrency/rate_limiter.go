package concurrency

import (
	"fmt"
	"sync"
	"time"

	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// RateLimiter manages token bucket rate limiters per function
type RateLimiter struct {
	mu      sync.RWMutex
	buckets map[string]*TokenBucket // "functionID" or "functionID:key" -> bucket
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{buckets: make(map[string]*TokenBucket)}
}

// Acquire waits (or rejects) until a token is available for the given function.
// Returns nil on success, or an error if the reject strategy is active and no token is available.
func (rl *RateLimiter) Acquire(functionID string, config workflow.RateLimitConfig, keyValue string) error {
	bucketKey := functionID
	if config.Key != "" && keyValue != "" {
		bucketKey = fmt.Sprintf("%s:%s", functionID, keyValue)
	}

	bucket := rl.getOrCreate(bucketKey, config.Limit, config.Period.TimeDuration())

	if config.Strategy == workflow.RateLimitReject {
		wait := bucket.Take()
		if wait > 0 {
			return fmt.Errorf("rate limit exceeded for %s, retry after %v", functionID, wait)
		}
		return nil
	}

	// Default: queue (block until token available)
	bucket.Wait()
	return nil
}

func (rl *RateLimiter) getOrCreate(key string, limit int, period time.Duration) *TokenBucket {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	bucket, exists := rl.buckets[key]
	if !exists {
		bucket = NewTokenBucket(limit, period)
		rl.buckets[key] = bucket
	}
	return bucket
}
