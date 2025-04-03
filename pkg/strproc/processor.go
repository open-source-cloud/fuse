// Package strproc provides string processing operations for workflow nodes.
package strproc

import (
	"context"
	"fmt"
	"strings"

	"github.com/open-source-cloud/fuse/internal/workflow"
)

const (
	// OperationUppercase represents the uppercase string operation
	OperationUppercase = "uppercase"
	// OperationLowercase represents the lowercase string operation
	OperationLowercase = "lowercase"
	// OperationTrim represents the trim string operation
	OperationTrim = "trim"
)

// StringProcessorConfig represents the configuration for a string processor node
type StringProcessorConfig struct {
	Operation string // One of OperationUppercase, OperationLowercase, OperationTrim
	Input     string // The input string to process
}

// StringProcessorNode implements the workflow.Node interface
type StringProcessorNode struct {
	id     string
	config StringProcessorConfig
}

// NewStringProcessorProvider creates a new provider for string processing nodes
func NewStringProcessorProvider() workflow.NodeProvider {
	return &stringProcessorProvider{}
}

type stringProcessorProvider struct{}

func (p *stringProcessorProvider) Name() string {
	return "string_processor"
}

func (p *stringProcessorProvider) CreateNode(config interface{}) (workflow.Node, error) {
	cfg, ok := config.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid config type for string processor")
	}

	operation, ok := cfg["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation is required for string processor")
	}

	input, ok := cfg["input"].(string)
	if !ok {
		return nil, fmt.Errorf("input is required for string processor")
	}

	return &StringProcessorNode{
		id: fmt.Sprintf("string_processor_%s", operation),
		config: StringProcessorConfig{
			Operation: operation,
			Input:     input,
		},
	}, nil
}

func (p *stringProcessorProvider) ValidateConfig(config interface{}) error {
	cfg, ok := config.(StringProcessorConfig)
	if !ok {
		return fmt.Errorf("invalid config type for string processor")
	}

	switch cfg.Operation {
	case OperationUppercase, OperationLowercase, OperationTrim:
		return nil
	default:
		return fmt.Errorf("invalid operation: %s", cfg.Operation)
	}
}

// ID returns the unique identifier of the string processor node
func (n *StringProcessorNode) ID() string {
	return n.id
}

// Execute processes the input string according to the configured operation
func (n *StringProcessorNode) Execute(_ context.Context, input interface{}) (interface{}, error) {
	str := n.config.Input
	if input != nil {
		// If input is provided, use it instead of config input
		if s, ok := input.(string); ok {
			str = s
		}
	}

	switch n.config.Operation {
	case OperationUppercase:
		return strings.ToUpper(str), nil
	case OperationLowercase:
		return strings.ToLower(str), nil
	case OperationTrim:
		return strings.TrimSpace(str), nil
	default:
		return nil, fmt.Errorf("unsupported operation: %s", n.config.Operation)
	}
}

// Validate checks if the string processor node is properly configured
func (n *StringProcessorNode) Validate() error {
	switch n.config.Operation {
	case OperationUppercase, OperationLowercase, OperationTrim:
		return nil
	default:
		return fmt.Errorf("invalid operation: %s", n.config.Operation)
	}
}
