package workflow

import (
	"context"
)

// Node represents a single step in the workflow
type Node interface {
	// ID returns the unique identifier of the node
	ID() string
	// Execute performs the node's logic
	Execute(ctx context.Context, input interface{}) (interface{}, error)
	// Validate checks if the node's configuration is valid
	Validate() error
}

// NodeProvider is responsible for creating and managing nodes
type NodeProvider interface {
	// Name returns the unique name of the provider
	Name() string
	// CreateNode creates a new node instance with the given configuration
	CreateNode(config interface{}) (Node, error)
	// ValidateConfig validates the configuration for a node
	ValidateConfig(config interface{}) error
}
