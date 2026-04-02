package workflow

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetryPolicy_DelayFor_Fixed(t *testing.T) {
	policy := RetryPolicy{
		MaxAttempts: 3,
		Backoff: BackoffConfig{
			Type:            BackoffFixed,
			InitialInterval: 500 * time.Millisecond,
		},
	}

	assert.Equal(t, 500*time.Millisecond, policy.DelayFor(0))
	assert.Equal(t, 500*time.Millisecond, policy.DelayFor(1))
	assert.Equal(t, 500*time.Millisecond, policy.DelayFor(5))
}

func TestRetryPolicy_DelayFor_Exponential(t *testing.T) {
	policy := RetryPolicy{
		MaxAttempts: 5,
		Backoff: BackoffConfig{
			Type:            BackoffExponential,
			InitialInterval: 1 * time.Second,
			MaxInterval:     30 * time.Second,
			Multiplier:      2.0,
		},
	}

	assert.Equal(t, 1*time.Second, policy.DelayFor(0))   // 1 * 2^0 = 1s
	assert.Equal(t, 2*time.Second, policy.DelayFor(1))   // 1 * 2^1 = 2s
	assert.Equal(t, 4*time.Second, policy.DelayFor(2))   // 1 * 2^2 = 4s
	assert.Equal(t, 8*time.Second, policy.DelayFor(3))   // 1 * 2^3 = 8s
	assert.Equal(t, 16*time.Second, policy.DelayFor(4))  // 1 * 2^4 = 16s
	assert.Equal(t, 30*time.Second, policy.DelayFor(10)) // capped at MaxInterval
}

func TestRetryPolicy_DelayFor_Linear(t *testing.T) {
	policy := RetryPolicy{
		MaxAttempts: 5,
		Backoff: BackoffConfig{
			Type:            BackoffLinear,
			InitialInterval: 1 * time.Second,
			MaxInterval:     5 * time.Second,
		},
	}

	assert.Equal(t, 1*time.Second, policy.DelayFor(0)) // 1 * (0+1) = 1s
	assert.Equal(t, 2*time.Second, policy.DelayFor(1)) // 1 * (1+1) = 2s
	assert.Equal(t, 3*time.Second, policy.DelayFor(2)) // 1 * (2+1) = 3s
	assert.Equal(t, 5*time.Second, policy.DelayFor(5)) // capped at MaxInterval
}

func TestRetryPolicy_DelayFor_ExponentialRespectsMaxInterval(t *testing.T) {
	policy := RetryPolicy{
		MaxAttempts: 10,
		Backoff: BackoffConfig{
			Type:            BackoffExponential,
			InitialInterval: 100 * time.Millisecond,
			MaxInterval:     1 * time.Second,
			Multiplier:      3.0,
		},
	}

	// 100ms * 3^4 = 8.1s, should be capped at 1s
	assert.Equal(t, 1*time.Second, policy.DelayFor(4))
}

func TestRetryTracker(t *testing.T) {
	tracker := NewRetryTracker()

	// Initially zero
	assert.Equal(t, 0, tracker.GetAttempts("exec-1"))

	// Increment
	assert.Equal(t, 1, tracker.Increment("exec-1"))
	assert.Equal(t, 2, tracker.Increment("exec-1"))
	assert.Equal(t, 2, tracker.GetAttempts("exec-1"))

	// Clear
	tracker.Clear("exec-1")
	assert.Equal(t, 0, tracker.GetAttempts("exec-1"))

	// Different exec IDs are independent
	tracker.Increment("exec-a")
	tracker.Increment("exec-b")
	assert.Equal(t, 1, tracker.GetAttempts("exec-a"))
	assert.Equal(t, 1, tracker.GetAttempts("exec-b"))
}

func TestDefaultRetryPolicy(t *testing.T) {
	policy := DefaultRetryPolicy()

	assert.Equal(t, 3, policy.MaxAttempts)
	assert.Equal(t, BackoffExponential, policy.Backoff.Type)
	assert.Equal(t, 1*time.Second, policy.Backoff.InitialInterval)
	assert.Equal(t, 30*time.Second, policy.Backoff.MaxInterval)
	assert.Equal(t, 2.0, policy.Backoff.Multiplier)
}
