package concurrency

import (
	"testing"
	"time"

	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_QueueStrategy(t *testing.T) {
	rl := NewRateLimiter()
	config := workflow.RateLimitConfig{
		Limit:    1,
		Period:   workflow.Duration(100 * time.Millisecond),
		Strategy: workflow.RateLimitQueue,
	}

	// First should succeed immediately
	start := time.Now()
	err := rl.Acquire("fn-a", config, "")
	assert.NoError(t, err)

	// Second should block until refill
	err = rl.Acquire("fn-a", config, "")
	assert.NoError(t, err)
	elapsed := time.Since(start)

	assert.GreaterOrEqual(t, elapsed, 80*time.Millisecond) // Allow tolerance
}

func TestRateLimiter_RejectStrategy(t *testing.T) {
	rl := NewRateLimiter()
	config := workflow.RateLimitConfig{
		Limit:    1,
		Period:   workflow.Duration(1 * time.Second),
		Strategy: workflow.RateLimitReject,
	}

	// First should succeed
	err := rl.Acquire("fn-b", config, "")
	assert.NoError(t, err)

	// Second should be rejected
	err = rl.Acquire("fn-b", config, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rate limit exceeded")
}

func TestRateLimiter_PerKeyIsolation(t *testing.T) {
	rl := NewRateLimiter()
	config := workflow.RateLimitConfig{
		Limit:    1,
		Period:   workflow.Duration(1 * time.Second),
		Key:      "input.apiKey",
		Strategy: workflow.RateLimitReject,
	}

	// Different keys should not conflict
	err := rl.Acquire("fn-c", config, "key-1")
	assert.NoError(t, err)

	err = rl.Acquire("fn-c", config, "key-2")
	assert.NoError(t, err)
}
