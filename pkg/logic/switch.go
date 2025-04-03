package logic

import (
	"context"
	"fmt"

	"github.com/open-source-cloud/fuse/internal/workflow"
)

// Case represents a single case in a switch statement
type Case struct {
	Value     interface{}
	Result    interface{}
	IsDefault bool
}

// SwitchConfig represents the configuration for a switch condition node
type SwitchConfig struct {
	Cases []Case
}

// SwitchNode implements the workflow.Node interface for switch conditions
type SwitchNode struct {
	*ConditionNode
	switchConfig SwitchConfig
}

// NewSwitchProvider creates a new provider for switch condition nodes
func NewSwitchProvider() workflow.NodeProvider {
	return &switchProvider{}
}

type switchProvider struct{}

func (p *switchProvider) Name() string {
	return "switch"
}

func (p *switchProvider) CreateNode(config interface{}) (workflow.Node, error) {
	cfg, ok := config.(SwitchConfig)
	if !ok {
		return nil, fmt.Errorf("invalid config type for switch condition")
	}

	baseNode, err := NewConditionProvider().CreateNode(ConditionConfig{})
	if err != nil {
		return nil, err
	}

	return &SwitchNode{
		ConditionNode: baseNode.(*ConditionNode),
		switchConfig:  cfg,
	}, nil
}

func (p *switchProvider) ValidateConfig(config interface{}) error {
	cfg, ok := config.(SwitchConfig)
	if !ok {
		return fmt.Errorf("invalid config type for switch condition")
	}

	if len(cfg.Cases) == 0 {
		return fmt.Errorf("switch must have at least one case")
	}

	hasDefault := false
	for _, c := range cfg.Cases {
		if c.IsDefault {
			if hasDefault {
				return fmt.Errorf("switch can only have one default case")
			}
			hasDefault = true
		}
	}

	return nil
}

// Execute evaluates the switch condition and returns the result
func (n *SwitchNode) Execute(_ context.Context, input interface{}) (interface{}, error) {
	// First try to match a specific case
	str, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("switch node requires string input")
	}

	for _, c := range n.switchConfig.Cases {
		if c.Value == str {
			return c.Result, nil
		}
	}

	// If no case matches, look for a default case
	for _, c := range n.switchConfig.Cases {
		if c.IsDefault {
			return c.Result, nil
		}
	}

	return nil, fmt.Errorf("no matching case found")
}
