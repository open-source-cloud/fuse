package logic

import (
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// SumFunctionID is the id of the sum function
const SumFunctionID = "sum"

// SumFunctionMetadata returns the metadata of the sum function
func SumFunctionMetadata() workflow.FunctionMetadata {
	return workflow.FunctionMetadata{
		Input: workflow.InputMetadata{
			Parameters: []workflow.ParameterSchema{
				{
					Name:        "values",
					Type:        "[]float64",
					Required:    true,
					Validations: nil,
					Description: "Values to sum",
					Default:     []int{},
				},
			},
		},
		Output: workflow.OutputMetadata{
			Parameters: []workflow.ParameterSchema{
				{
					Name:        "sum",
					Type:        "float64",
					Validations: nil,
					Description: "Result of the sum",
					Default:     0,
				},
			},
		},
	}
}

// SumFunction executes the sum function and returns the sum of the values
func SumFunction(execInfo *workflow.ExecutionInfo) (workflow.FunctionResult, error) {
	sum := float64(0)
	values := execInfo.Input.GetFloat64SliceOrDefault("values", []float64{})

	for _, value := range values {
		sum += value
	}

	return workflow.NewFunctionResult(workflow.FunctionSuccess, map[string]any{"sum": sum}), nil
}
