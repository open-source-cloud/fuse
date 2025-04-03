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

// Workflow represents a complete workflow definition
type Workflow struct {
	ID          string
	Name        string
	Description string
	Nodes       []Node
	Edges       []Edge
}

// Edge represents a connection between nodes
type Edge struct {
	FromNodeID string
	ToNodeID   string
	Condition  func(interface{}) bool // Optional condition for edge traversal
}

// Engine represents the workflow execution engine
type Engine interface {
	// RegisterProvider registers a new node provider
	RegisterProvider(provider NodeProvider) error
	// ExecuteWorkflow runs a workflow with the given input
	ExecuteWorkflow(ctx context.Context, workflow *Workflow, input interface{}) (interface{}, error)
	// ValidateWorkflow checks if a workflow definition is valid
	ValidateWorkflow(workflow *Workflow) error
}

// ExecutionContext holds the state during workflow execution
type ExecutionContext struct {
	WorkflowID string
	NodeID     string
	Input      interface{}
	Output     interface{}
	Error      error
}

// Error represents a workflow execution error
type Error struct {
	WorkflowID string
	NodeID     string
	Message    string
	Err        error
}

func (e *Error) Error() string {
	return e.Message
}
