package workflow

// ConcurrencyConfig defines concurrency limits for a workflow or function
type ConcurrencyConfig struct {
	// Limit is the maximum number of concurrent executions
	Limit int `json:"limit" validate:"min=1,max=1000"`
	// Key is an optional expression that scopes the limit (e.g., "input.userId").
	// When set, the limit applies per unique key value.
	Key string `json:"key,omitempty"`
}
