// Package workflow provides the core workflow engine implementation.
// It includes types and functions for defining and executing workflows.
package workflow

import (
	"context"
	"fmt"
	"sync"
)

// Engine represents the workflow execution engine
type Engine interface {
	// RegisterProvider registers a new node provider
	RegisterProvider(provider NodeProvider) error
	// ExecuteWorkflow runs a workflow with the given input
	ExecuteWorkflow(ctx context.Context, workflow *Workflow, input interface{}) (interface{}, error)
	// ValidateWorkflow checks if a workflow definition is valid
	ValidateWorkflow(workflow *Workflow) error
}

// DefaultEngine implements the WorkflowEngine interface
type DefaultEngine struct {
	providers map[string]NodeProvider
	mu        sync.RWMutex
}

// NewDefaultEngine creates a new instance of DefaultEngine
func NewDefaultEngine() *DefaultEngine {
	return &DefaultEngine{
		providers: make(map[string]NodeProvider),
	}
}

// RegisterProvider implements WorkflowEngine.RegisterProvider
func (e *DefaultEngine) RegisterProvider(provider NodeProvider) error {
	if provider == nil {
		return &Error{
			Message: "provider cannot be nil",
		}
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	name := provider.Name()
	if _, exists := e.providers[name]; exists {
		return fmt.Errorf("provider %s already registered", name)
	}

	e.providers[name] = provider
	return nil
}

// ExecuteWorkflow implements WorkflowEngine.ExecuteWorkflow
func (e *DefaultEngine) ExecuteWorkflow(ctx context.Context, workflow *Workflow, input interface{}) (interface{}, error) {
	if workflow == nil {
		return nil, &Error{
			Message: "workflow cannot be nil",
		}
	}
	if err := e.ValidateWorkflow(workflow); err != nil {
		return nil, err
	}

	// Create a map of nodes for faster lookup
	nodeMap := make(map[string]Node)
	for _, node := range workflow.Nodes {
		nodeMap[node.ID()] = node
	}

	// Execute the workflow
	currentInput := input
	for _, edge := range workflow.Edges {
		fromNode, exists := nodeMap[edge.FromNodeID]
		if !exists {
			return nil, &Error{
				Message: fmt.Sprintf("node %s not found", edge.FromNodeID),
			}
		}

		output, err := fromNode.Execute(ctx, currentInput)
		if err != nil {
			return nil, &Error{
				Message: fmt.Sprintf("error executing node %s: %v", edge.FromNodeID, err),
				Err:     err,
			}
		}

		// Check edge condition if present
		if edge.Condition != nil && !edge.Condition(output) {
			return nil, &Error{
				Message: fmt.Sprintf("edge condition not met for node %s", edge.FromNodeID),
			}
		}

		currentInput = output
	}

	return currentInput, nil
}

// ValidateWorkflow implements WorkflowEngine.ValidateWorkflow
func (e *DefaultEngine) ValidateWorkflow(workflow *Workflow) error {
	if workflow == nil {
		return fmt.Errorf("workflow is nil")
	}

	if workflow.ID == "" {
		return fmt.Errorf("workflow ID is empty")
	}

	if len(workflow.Nodes) == 0 {
		return fmt.Errorf("workflow has no nodes")
	}

	// Validate each node
	for _, node := range workflow.Nodes {
		if err := node.Validate(); err != nil {
			return &Error{
				Message: fmt.Sprintf("invalid node %s: %v", node.ID(), err),
				Err:     err,
			}
		}
	}

	// Validate edges
	nodeIDs := make(map[string]bool)
	for _, node := range workflow.Nodes {
		nodeIDs[node.ID()] = true
	}

	for _, edge := range workflow.Edges {
		if !nodeIDs[edge.FromNodeID] {
			return fmt.Errorf("edge references non-existent node %s", edge.FromNodeID)
		}
		if !nodeIDs[edge.ToNodeID] {
			return fmt.Errorf("edge references non-existent node %s", edge.ToNodeID)
		}
	}

	return nil
}
