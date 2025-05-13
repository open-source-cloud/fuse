package logic

import (
	"fmt"
	"github.com/expr-lang/expr"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

const IfFunctionID = "if"

// IfFunctionMetadata returns the metadata of the if function
func IfFunctionMetadata() workflow.FunctionMetadata {
	return workflow.NewFunctionMetadata(
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
		},
		// output
		workflow.OutputMetadata{
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

// IfFunction executes the if function and returns the result
func IfFunction(input *workflow.FunctionInput) (workflow.FunctionResult, error) {
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

	return workflow.NewFunctionResult(workflow.FunctionSuccess, map[string]any{"result": result}), nil
}
