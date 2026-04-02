package workflow

import (
	"math"
	"time"
)

// BackoffType defines the backoff strategy
type BackoffType string

const (
	// BackoffFixed constant delay between retries
	BackoffFixed BackoffType = "fixed"
	// BackoffExponential exponentially increasing delay
	BackoffExponential BackoffType = "exponential"
	// BackoffLinear linearly increasing delay
	BackoffLinear BackoffType = "linear"
)

// RetryPolicy defines how a node should handle failures
type RetryPolicy struct {
	// MaxAttempts is the maximum number of retry attempts (0 = no retries)
	MaxAttempts int `json:"maxAttempts" validate:"min=0,max=100"`
	// Backoff strategy
	Backoff BackoffConfig `json:"backoff"`
}

// BackoffConfig defines the backoff parameters
type BackoffConfig struct {
	// Type of backoff: fixed, exponential, linear
	Type BackoffType `json:"type" validate:"oneof=fixed exponential linear"`
	// InitialInterval is the base delay between retries
	InitialInterval FlexibleDuration `json:"initialInterval"`
	// MaxInterval caps the delay for exponential/linear backoff
	MaxInterval FlexibleDuration `json:"maxInterval"`
	// Multiplier for exponential backoff (default: 2.0)
	Multiplier float64 `json:"multiplier,omitempty"`
}

// DefaultRetryPolicy returns a sensible default (3 attempts, exponential backoff)
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: 3,
		Backoff: BackoffConfig{
			Type:            BackoffExponential,
			InitialInterval: FlexibleDuration(1 * time.Second),
			MaxInterval:     FlexibleDuration(30 * time.Second),
			Multiplier:      2.0,
		},
	}
}

// DelayFor calculates the delay for a given attempt number (0-indexed)
func (p RetryPolicy) DelayFor(attempt int) time.Duration {
	initial := time.Duration(p.Backoff.InitialInterval)
	maxInt := time.Duration(p.Backoff.MaxInterval)
	switch p.Backoff.Type {
	case BackoffExponential:
		delay := float64(initial) * math.Pow(p.Backoff.Multiplier, float64(attempt))
		if time.Duration(delay) > maxInt {
			return maxInt
		}
		return time.Duration(delay)
	case BackoffLinear:
		delay := initial * time.Duration(attempt+1)
		if delay > maxInt {
			return maxInt
		}
		return delay
	default: // fixed
		return initial
	}
}
