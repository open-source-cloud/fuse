// Package logic provides logical operators and conditions for workflow nodes.
package logic

import (
	"context"
	"fmt"

	"github.com/gustavobertoi/core-workflow-poc/internal/workflow"
)

// ConditionConfig represents the base configuration for condition nodes
type ConditionConfig struct {
	Value interface{}
}

// ConditionNode represents a node that evaluates a condition
type ConditionNode struct {
	id     string
	config ConditionConfig
}

// NewConditionProvider creates a new provider for condition nodes
func NewConditionProvider() workflow.NodeProvider {
	return &conditionProvider{}
}

type conditionProvider struct{}

func (p *conditionProvider) Name() string {
	return "condition"
}

func (p *conditionProvider) CreateNode(config interface{}) (workflow.Node, error) {
	cfg, ok := config.(ConditionConfig)
	if !ok {
		return nil, fmt.Errorf("invalid config type for condition")
	}

	return &ConditionNode{
		id:     fmt.Sprintf("condition_%v", cfg.Value),
		config: cfg,
	}, nil
}

func (p *conditionProvider) ValidateConfig(config interface{}) error {
	_, ok := config.(ConditionConfig)
	if !ok {
		return fmt.Errorf("invalid config type for condition")
	}
	return nil
}

// ID returns the unique identifier of the condition node
func (n *ConditionNode) ID() string {
	return n.id
}

// Validate checks if the condition node is properly configured
func (n *ConditionNode) Validate() error {
	return nil
}

// Execute evaluates the condition and returns the result
func (n *ConditionNode) Execute(_ context.Context, input interface{}) (interface{}, error) {
	// Base condition node just passes through the input
	return input, nil
}
