package logic

import (
	"context"
	"fmt"

	"github.com/gustavobertoi/core-workflow-poc/internal/workflow"
)

// IfConfig represents the configuration for an if condition node
type IfConfig struct {
	Condition func(interface{}) bool
}

// IfNode implements the workflow.Node interface for if conditions
type IfNode struct {
	*ConditionNode
	ifConfig IfConfig
}

// NewIfProvider creates a new provider for if condition nodes
func NewIfProvider() workflow.NodeProvider {
	return &ifProvider{}
}

type ifProvider struct{}

func (p *ifProvider) Name() string {
	return "if"
}

func (p *ifProvider) CreateNode(config interface{}) (workflow.Node, error) {
	cfg, ok := config.(IfConfig)
	if !ok {
		return nil, fmt.Errorf("invalid config type for if condition")
	}

	baseNode, err := NewConditionProvider().CreateNode(ConditionConfig{})
	if err != nil {
		return nil, err
	}

	return &IfNode{
		ConditionNode: baseNode.(*ConditionNode),
		ifConfig:      cfg,
	}, nil
}

func (p *ifProvider) ValidateConfig(config interface{}) error {
	cfg, ok := config.(IfConfig)
	if !ok {
		return fmt.Errorf("invalid config type for if condition")
	}
	if cfg.Condition == nil {
		return fmt.Errorf("condition function cannot be nil")
	}
	return nil
}

// Execute evaluates the if condition and returns the result
func (n *IfNode) Execute(_ context.Context, input interface{}) (interface{}, error) {
	if n.ifConfig.Condition(input) {
		return true, nil
	}
	return false, nil
}
