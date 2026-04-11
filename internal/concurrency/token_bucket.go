package concurrency

import (
	"math"
	"sync"
	"time"
)

// TokenBucket implements a token bucket rate limiter
type TokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

// NewTokenBucket creates a new token bucket with the given limit per period
func NewTokenBucket(limit int, period time.Duration) *TokenBucket {
	rate := float64(limit) / period.Seconds()
	return &TokenBucket{
		tokens:     float64(limit),
		maxTokens:  float64(limit),
		refillRate: rate,
		lastRefill: time.Now(),
	}
}

// Take attempts to consume one token. Returns 0 if a token was available,
// or the duration to wait until the next token is available.
func (tb *TokenBucket) Take() time.Duration {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens >= 1 {
		tb.tokens--
		return 0
	}

	// Calculate wait time until next token
	deficit := 1 - tb.tokens
	waitTime := time.Duration(deficit / tb.refillRate * float64(time.Second))
	return waitTime
}

// Wait blocks until a token is available and consumes it
func (tb *TokenBucket) Wait() {
	for {
		wait := tb.Take()
		if wait == 0 {
			return
		}
		time.Sleep(wait)
	}
}

func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens = math.Min(tb.maxTokens, tb.tokens+elapsed*tb.refillRate)
	tb.lastRefill = now
}
