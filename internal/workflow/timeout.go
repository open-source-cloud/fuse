package workflow

import "time"

// TimeoutConfig defines timeout settings for a node's function execution
type TimeoutConfig struct {
	// Execution timeout for this node's function.
	// Zero means no timeout (not recommended for external functions).
	Execution time.Duration `json:"execution,omitempty"`
}

// GraphTimeoutConfig defines timeout settings at the workflow level
type GraphTimeoutConfig struct {
	// Total maximum duration for the entire workflow execution.
	// Zero means no timeout.
	Total time.Duration `json:"total,omitempty"`
}
