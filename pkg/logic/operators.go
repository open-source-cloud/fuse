package logic

import (
	"context"
	"fmt"

	"github.com/gustavobertoi/core-workflow-poc/internal/workflow"
)

const (
	// NodeTypeAnd represents the type identifier for AND operator nodes
	NodeTypeAnd = "and"
	// NodeTypeOr represents the type identifier for OR operator nodes
	NodeTypeOr = "or"
	// NodeTypeSwitch represents the type identifier for switch nodes
	NodeTypeSwitch = "switch"
)

// OperatorConfig represents the configuration for logical operator nodes
type OperatorConfig struct {
	Values []interface{}
}

// AndNode implements the workflow.Node interface for logical AND operations
type AndNode struct {
	*ConditionNode
	operatorConfig OperatorConfig
}

// OrNode implements the workflow.Node interface for logical OR operations
type OrNode struct {
	*ConditionNode
	operatorConfig OperatorConfig
}

// NewAndProvider creates a new provider for AND operator nodes
func NewAndProvider() workflow.NodeProvider {
	return &andProvider{}
}

// NewOrProvider creates a new provider for OR operator nodes
func NewOrProvider() workflow.NodeProvider {
	return &orProvider{}
}

type andProvider struct{}
type orProvider struct{}

func (p *andProvider) Name() string {
	return NodeTypeAnd
}

func (p *orProvider) Name() string {
	return NodeTypeOr
}

func (p *andProvider) CreateNode(config interface{}) (workflow.Node, error) {
	cfg, ok := config.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid config type for AND operator")
	}

	values, ok := cfg["values"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("values is required for AND operator")
	}

	if len(values) < 2 {
		return nil, fmt.Errorf("AND operator requires at least 2 values")
	}

	baseNode, err := NewConditionProvider().CreateNode(ConditionConfig{})
	if err != nil {
		return nil, err
	}

	return &AndNode{
		ConditionNode: baseNode.(*ConditionNode),
		operatorConfig: OperatorConfig{
			Values: values,
		},
	}, nil
}

func (p *orProvider) CreateNode(config interface{}) (workflow.Node, error) {
	cfg, ok := config.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid config type for OR operator")
	}

	values, ok := cfg["values"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("values is required for OR operator")
	}

	if len(values) < 2 {
		return nil, fmt.Errorf("OR operator requires at least 2 values")
	}

	baseNode, err := NewConditionProvider().CreateNode(ConditionConfig{})
	if err != nil {
		return nil, err
	}

	return &OrNode{
		ConditionNode: baseNode.(*ConditionNode),
		operatorConfig: OperatorConfig{
			Values: values,
		},
	}, nil
}

func (p *andProvider) ValidateConfig(config interface{}) error {
	cfg, ok := config.(OperatorConfig)
	if !ok {
		return fmt.Errorf("invalid config type for AND operator")
	}
	if len(cfg.Values) < 2 {
		return fmt.Errorf("AND operator requires at least 2 values")
	}
	return nil
}

func (p *orProvider) ValidateConfig(config interface{}) error {
	cfg, ok := config.(OperatorConfig)
	if !ok {
		return fmt.Errorf("invalid config type for OR operator")
	}
	if len(cfg.Values) < 2 {
		return fmt.Errorf("OR operator requires at least 2 values")
	}
	return nil
}

// Execute evaluates the AND condition and returns the result
func (n *AndNode) Execute(_ context.Context, input interface{}) (interface{}, error) {
	// Convert all values to boolean
	for _, v := range n.operatorConfig.Values {
		boolVal, ok := v.(bool)
		if !ok {
			return nil, fmt.Errorf("AND operator requires boolean values")
		}
		if !boolVal {
			return false, nil
		}
	}

	// Check input value
	boolVal, ok := input.(bool)
	if !ok {
		return nil, fmt.Errorf("AND operator requires boolean values")
	}
	if !boolVal {
		return false, nil
	}

	return true, nil
}

// Execute evaluates the OR condition and returns the result
func (n *OrNode) Execute(_ context.Context, input interface{}) (interface{}, error) {
	// Convert all values to boolean
	for _, v := range n.operatorConfig.Values {
		boolVal, ok := v.(bool)
		if !ok {
			return nil, fmt.Errorf("OR operator requires boolean values")
		}
		if boolVal {
			return true, nil
		}
	}

	// Check input value
	boolVal, ok := input.(bool)
	if !ok {
		return nil, fmt.Errorf("OR operator requires boolean values")
	}
	if boolVal {
		return true, nil
	}

	return false, nil
}

// ID returns the unique identifier of the AND node
func (n *AndNode) ID() string {
	return NodeTypeAnd
}

// ID returns the unique identifier of the OR node
func (n *OrNode) ID() string {
	return NodeTypeOr
}

// ID returns the unique identifier of the switch node
func (n *SwitchNode) ID() string {
	return NodeTypeSwitch
}
