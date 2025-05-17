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
			Parameters: workflow.Parameters{
				"values": workflow.ParameterSchema{
					Name:        "values",
					Type:        "[]int",
					Required:    true,
					Validations: nil,
					Description: "Values to sum",
					Default:     []int{},
				},
			},
		},
		Output: workflow.OutputMetadata{
			Parameters: workflow.Parameters{
				"result": workflow.ParameterSchema{
					Name:        "sum",
					Type:        "int",
					Validations: nil,
					Description: "Result of the sum",
					Default:     0,
				},
			},
		},
	}
}

// SumFunction executes the sum function and returns the sum of the values
func SumFunction(input *workflow.FunctionInput) (workflow.FunctionResult, error) {
	sum := 0
	values := input.GetIntSliceOrDefault("values", []int{})

	for _, value := range values {
		sum += value
	}

	return workflow.NewFunctionResult(workflow.FunctionSuccess, map[string]any{"sum": sum}), nil
}
