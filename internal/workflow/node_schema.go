package workflow

type (
	// NodeSchema represents a structure containing an ID and the function reference for this node.
	NodeSchema struct {
		ID       string         `json:"id" yaml:"id" validate:"required"`
		Function string         `json:"function" yaml:"function" validate:"required"`
		Retry    *RetryPolicy   `json:"retry,omitempty" yaml:"retry,omitempty"`
		Timeout  *TimeoutConfig `json:"timeout,omitempty" yaml:"timeout,omitempty"`
		Merge    *MergeConfig   `json:"merge,omitempty" yaml:"merge,omitempty"`
	}
)

// Clone creates a deep copy of the NodeSchema
func (n *NodeSchema) Clone() *NodeSchema {
	clone := &NodeSchema{
		ID:       n.ID,
		Function: n.Function,
	}
	if n.Retry != nil {
		r := *n.Retry
		clone.Retry = &r
	}
	if n.Timeout != nil {
		t := *n.Timeout
		clone.Timeout = &t
	}
	if n.Merge != nil {
		m := *n.Merge
		clone.Merge = &m
	}
	return clone
}
