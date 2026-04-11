package workflow

import (
	"encoding/json"
	"fmt"
	"time"
)

// RateLimitStrategy defines behavior when the rate limit is exceeded
type RateLimitStrategy string

const (
	// RateLimitQueue queues excess requests until a token is available (default)
	RateLimitQueue RateLimitStrategy = "queue"
	// RateLimitReject rejects excess requests immediately
	RateLimitReject RateLimitStrategy = "reject"
)

// RateLimitConfig defines rate limiting parameters for a function
type RateLimitConfig struct {
	// Limit is the maximum number of executions allowed per period
	Limit int `json:"limit" validate:"min=1"`
	// Period is the time window for the rate limit (e.g., "1m", "1h")
	Period Duration `json:"period"`
	// Key is an optional expression to scope the rate limit (e.g., "input.apiKey")
	Key string `json:"key,omitempty"`
	// Strategy defines behavior when limit is exceeded (default: queue)
	Strategy RateLimitStrategy `json:"strategy,omitempty"`
}

// Duration is a time.Duration that JSON-encodes as a Go duration string
type Duration time.Duration

// TimeDuration returns the underlying time.Duration
func (d Duration) TimeDuration() time.Duration {
	return time.Duration(d)
}

// MarshalJSON encodes as a duration string
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalJSON decodes from a JSON string or integer nanoseconds
func (d *Duration) UnmarshalJSON(data []byte) error {
	if d == nil {
		return fmt.Errorf("Duration: UnmarshalJSON on nil pointer")
	}
	if len(data) > 0 && data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		parsed, err := time.ParseDuration(s)
		if err != nil {
			return err
		}
		*d = Duration(parsed)
		return nil
	}
	var ns int64
	if err := json.Unmarshal(data, &ns); err != nil {
		return err
	}
	*d = Duration(time.Duration(ns))
	return nil
}
