package logic

import (
	"context"
	"fmt"

	"github.com/open-source-cloud/fuse/internal/workflow"
)

const (
	// NodeTypeElse represents the type identifier for else condition nodes
	NodeTypeElse = "else"
)

// ElseConfig represents the configuration for an else condition node
type ElseConfig struct {
	DefaultValue interface{}
}

// ElseNode implements the workflow.GraphNode interface for else conditions
type ElseNode struct {
	*ConditionNode
	elseConfig ElseConfig
}

// NewElseProvider creates a new provider for else condition nodes
func NewElseProvider() workflow.NodeProvider {
	return &elseProvider{}
}

type elseProvider struct{}

func (p *elseProvider) Name() string {
	return NodeTypeElse
}

func (p *elseProvider) CreateNode(config interface{}) (workflow.Node, error) {
	cfg, ok := config.(ElseConfig)
	if !ok {
		return nil, fmt.Errorf("invalid config type for else condition")
	}

	baseNode, err := NewConditionProvider().CreateNode(ConditionConfig{})
	if err != nil {
		return nil, err
	}

	return &ElseNode{
		ConditionNode: baseNode.(*ConditionNode),
		elseConfig:    cfg,
	}, nil
}

func (p *elseProvider) ValidateConfig(config interface{}) error {
	_, ok := config.(ElseConfig)
	if !ok {
		return fmt.Errorf("invalid config type for else condition")
	}
	return nil
}

// Execute evaluates the else condition and returns the result
func (n *ElseNode) Execute(_ context.Context, input interface{}) (interface{}, error) {
	// If input is false (from if condition), return the default value
	if input == false {
		return n.elseConfig.DefaultValue, nil
	}
	return input, nil
}

// ID returns the unique identifier of the else node
func (n *ElseNode) ID() string {
	return NodeTypeElse
}
