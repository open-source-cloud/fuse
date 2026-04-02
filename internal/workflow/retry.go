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
	MaxAttempts int `json:"maxAttempts" bson:"maxAttempts" validate:"min=0,max=100"`
	// Backoff strategy
	Backoff BackoffConfig `json:"backoff" bson:"backoff"`
}

// BackoffConfig defines the backoff parameters
type BackoffConfig struct {
	// Type of backoff: fixed, exponential, linear
	Type BackoffType `json:"type" bson:"type" validate:"oneof=fixed exponential linear"`
	// InitialInterval is the base delay between retries
	InitialInterval time.Duration `json:"initialInterval" bson:"initialInterval"`
	// MaxInterval caps the delay for exponential/linear backoff
	MaxInterval time.Duration `json:"maxInterval" bson:"maxInterval"`
	// Multiplier for exponential backoff (default: 2.0)
	Multiplier float64 `json:"multiplier,omitempty" bson:"multiplier,omitempty"`
}

// DefaultRetryPolicy returns a sensible default (3 attempts, exponential backoff)
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: 3,
		Backoff: BackoffConfig{
			Type:            BackoffExponential,
			InitialInterval: 1 * time.Second,
			MaxInterval:     30 * time.Second,
			Multiplier:      2.0,
		},
	}
}

// DelayFor calculates the delay for a given attempt number (0-indexed)
func (p RetryPolicy) DelayFor(attempt int) time.Duration {
	switch p.Backoff.Type {
	case BackoffExponential:
		delay := float64(p.Backoff.InitialInterval) * math.Pow(p.Backoff.Multiplier, float64(attempt))
		if time.Duration(delay) > p.Backoff.MaxInterval {
			return p.Backoff.MaxInterval
		}
		return time.Duration(delay)
	case BackoffLinear:
		delay := p.Backoff.InitialInterval * time.Duration(attempt+1)
		if delay > p.Backoff.MaxInterval {
			return p.Backoff.MaxInterval
		}
		return delay
	default: // fixed
		return p.Backoff.InitialInterval
	}
}
