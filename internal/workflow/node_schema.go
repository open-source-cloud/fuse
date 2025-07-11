package workflow

type (
	// NodeSchema represents a structure containing an ID, package details, and optional configuration.
	NodeSchema struct {
		ID       string      `json:"id" yaml:"id" validate:"required"`
		Function string      `json:"function" yaml:"function" validate:"required"`
		Config   *NodeConfig `json:"config,omitempty" yaml:"config,omitempty"`
	}
	// NodeConfig represents the configuration schema for a node. TODO
	NodeConfig struct{}
)

// Clone creates a deep copy of the NodeSchema
func (n *NodeSchema) Clone() *NodeSchema {
	return &NodeSchema{
		ID:       n.ID,
		Function: n.Function,
		Config:   &NodeConfig{},
	}
}
