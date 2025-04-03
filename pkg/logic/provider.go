package logic

import (
	"fmt"

	"github.com/open-source-cloud/fuse/internal/workflow"
)

// ProcessorProvider provides logic processing nodes
type ProcessorProvider interface {
	workflow.NodeProvider
	CreateNode(config interface{}) (workflow.Node, error)
}

type logicProcessorProvider struct{}

// NewLogicProcessorProvider creates a new provider for logic processing nodes
func NewLogicProcessorProvider() ProcessorProvider {
	return &logicProcessorProvider{}
}

func (p *logicProcessorProvider) Name() string {
	return "logic"
}

func (p *logicProcessorProvider) CreateNode(config interface{}) (workflow.Node, error) {
	cfg, ok := config.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid config type for logic processor")
	}

	nodeType, ok := cfg["type"].(string)
	if !ok {
		return nil, fmt.Errorf("node type is required for logic processor")
	}

	switch nodeType {
	case NodeTypeAnd:
		return NewAndProvider().CreateNode(config)
	case NodeTypeOr:
		return NewOrProvider().CreateNode(config)
	case NodeTypeSwitch:
		return NewSwitchProvider().CreateNode(config)
	default:
		return nil, fmt.Errorf("unsupported node type: %s", nodeType)
	}
}

func (p *logicProcessorProvider) ValidateConfig(config interface{}) error {
	cfg, ok := config.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid config type for logic processor")
	}

	nodeType, ok := cfg["type"].(string)
	if !ok {
		return fmt.Errorf("node type is required for logic processor")
	}

	switch nodeType {
	case NodeTypeAnd, NodeTypeOr, NodeTypeSwitch:
		return nil
	default:
		return fmt.Errorf("unsupported node type: %s", nodeType)
	}
}
