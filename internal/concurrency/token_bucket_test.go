package concurrency

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTokenBucket_InitialTokens(t *testing.T) {
	tb := NewTokenBucket(5, 1*time.Second)

	// Should have 5 tokens available immediately
	for range 5 {
		wait := tb.Take()
		assert.Equal(t, time.Duration(0), wait)
	}

	// 6th should require waiting
	wait := tb.Take()
	assert.Greater(t, wait, time.Duration(0))
}

func TestTokenBucket_RefillOverTime(t *testing.T) {
	tb := NewTokenBucket(1, 100*time.Millisecond)

	// Consume the one available token
	wait := tb.Take()
	assert.Equal(t, time.Duration(0), wait)

	// Immediately, no token available
	wait = tb.Take()
	assert.Greater(t, wait, time.Duration(0))

	// Wait for refill
	time.Sleep(120 * time.Millisecond)

	// Should have a token now
	wait = tb.Take()
	assert.Equal(t, time.Duration(0), wait)
}

func TestTokenBucket_Wait(t *testing.T) {
	tb := NewTokenBucket(1, 50*time.Millisecond)

	// Consume the available token
	tb.Wait()

	// This should block until refill
	start := time.Now()
	tb.Wait()
	elapsed := time.Since(start)

	assert.GreaterOrEqual(t, elapsed, 40*time.Millisecond) // Allow some tolerance
}

func TestTokenBucket_BurstBehavior(t *testing.T) {
	tb := NewTokenBucket(10, 1*time.Second)

	// Should be able to burst 10 tokens at once
	for range 10 {
		wait := tb.Take()
		assert.Equal(t, time.Duration(0), wait)
	}

	// Next one should require waiting
	wait := tb.Take()
	assert.Greater(t, wait, time.Duration(0))
}
