package logic

import (
	"fmt"
	"github.com/expr-lang/expr"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// IfNodeID is the ID of the sum node
const IfNodeID = "fuse.io/workflows/internal/logic/if"

// IfNode is a sum node
type IfNode struct {
	workflow.Node
}

// NewIfNode creates a new sum node
func NewIfNode() workflow.Node {
	return &IfNode{}
}

// ID returns the ID of the sum node
func (n *IfNode) ID() string {
	return IfNodeID
}

// Metadata returns the metadata of the sum node
func (n *IfNode) Metadata() workflow.NodeMetadata {
	return workflow.NewNodeMetadata(
		// input
		workflow.InputMetadata{
			CustomParameters: true,
			Parameters: workflow.Parameters{
				"expression": workflow.ParameterSchema{
					Name:        "expression",
					Type:        "string",
					Required:    true,
					Validations: nil,
					Description: "Expression to evaluate",
					Default:     "",
				},
			},
			Edges: workflow.InputEdgeMetadata{},
		},
		// output
		workflow.OutputMetadata{
			Parameters:        workflow.Parameters{},
			ConditionalOutput: true,
			Edges: map[string]workflow.OutputEdgeMetadata{
				"condition-true": {
					Name: "condition-true",
					ConditionalEdge: workflow.ConditionalEdgeMetadata{
						Condition: "result",
						Value:     true,
					},
				},
				"condition-false": {
					Name: "condition-false",
					ConditionalEdge: workflow.ConditionalEdgeMetadata{
						Condition: "result",
						Value:     false,
					},
				},
			},
		},
	)
}

// Execute executes the sum node and returns the sum of the values
func (n *IfNode) Execute(input *workflow.NodeInput) (workflow.NodeResult, error) {
	exprStr := input.Get("expression").(string)
	if exprStr == "" {
		return nil, fmt.Errorf("expression is empty")
	}

	compiledExpr, err := expr.Compile(exprStr, expr.Env(input.Raw()))
	if err != nil {
		return nil, fmt.Errorf("failed to compile expression: %w", err)
	}

	result, err := expr.Run(compiledExpr, input.Raw())
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate expression: %w", err)
	}

	return workflow.NewNodeResult(workflow.NodeOutputStatusSuccess, map[string]any{"result": result}), nil
}
