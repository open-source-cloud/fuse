package workflow

type (
	// NodeSchema represents a structure containing an ID, package details, and optional configuration.
	NodeSchema struct {
		ID       string         `json:"id" yaml:"id" validate:"required"`
		Function string         `json:"function" yaml:"function" validate:"required"`
		Config   *NodeConfig    `json:"config,omitempty" yaml:"config,omitempty"`
		Retry    *RetryPolicy   `json:"retry,omitempty" yaml:"retry,omitempty"`
		Timeout  *TimeoutConfig `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	}
	// NodeConfig represents the configuration schema for a node. TODO
	NodeConfig struct{}
)

// Clone creates a deep copy of the NodeSchema
func (n *NodeSchema) Clone() *NodeSchema {
	clone := &NodeSchema{
		ID:       n.ID,
		Function: n.Function,
		Config:   &NodeConfig{},
	}
	if n.Retry != nil {
		r := *n.Retry
		clone.Retry = &r
	}
	if n.Timeout != nil {
		t := *n.Timeout
		clone.Timeout = &t
	}
	return clone
}
