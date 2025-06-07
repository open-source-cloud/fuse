package logic

import (
	"fmt"
	"github.com/expr-lang/expr"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// IfFunctionID if function ID
const IfFunctionID = "if"

// IfFunctionMetadata returns the metadata of the if function
func IfFunctionMetadata() workflow.FunctionMetadata {
	return workflow.FunctionMetadata{
		Input: workflow.InputMetadata{
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
		},
		Output: workflow.OutputMetadata{
			ConditionalOutput: true,
			ConditionalOutputField: "result",
			Edges: map[string]workflow.OutputEdgeMetadata{
				"if-true": {
					Name: "if-true",
					ConditionalEdge: workflow.ConditionalEdgeMetadata{
						Value:     true,
					},
				},
				"if-false": {
					Name: "if-false",
					ConditionalEdge: workflow.ConditionalEdgeMetadata{
						Value:     false,
					},
				},
			},
		},
	}
}

// IfFunction executes the if function and returns the result
func IfFunction(_ *workflow.ExecutionInfo, input *workflow.FunctionInput) (workflow.FunctionResult, error) {
	exprStr, ok := input.Get("expression").(string)
	if !ok || exprStr == "" {
		return workflow.NewFunctionResultError(fmt.Errorf("expression is empty"))
	}

	compiledExpr, err := expr.Compile(exprStr, expr.Env(input.Raw()))
	if err != nil {
		return workflow.NewFunctionResultError(fmt.Errorf("failed to compile expression: %w", err))
	}

	result, err := expr.Run(compiledExpr, input.Raw())
	if err != nil {
		return workflow.NewFunctionResultError(fmt.Errorf("failed to evaluate expression: %w", err))
	}

	return workflow.NewFunctionResultSuccessWith(map[string]any{"result": result}), nil
}
