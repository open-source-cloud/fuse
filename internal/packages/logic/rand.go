package logic

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/open-source-cloud/fuse/pkg/workflow"
)

const RandFunctionID = "rand"

// RandFunctionMetadata returns the metadata of the rand function
func RandFunctionMetadata() workflow.FunctionMetadata {
	return workflow.NewFunctionMetadata(
		// input
		workflow.InputMetadata{
			Parameters: workflow.Parameters{
				"min": workflow.ParameterSchema{
					Name:        "min",
					Type:        "int",
					Required:    false,
					Validations: nil,
					Description: "Minimum value of the random number",
					Default:     0,
				},
				"max": workflow.ParameterSchema{
					Name:        "max",
					Type:        "int",
					Required:    false,
					Validations: nil,
					Description: "Maximum value of the random number",
					Default:     100,
				},
			},
		},
		// output
		workflow.OutputMetadata{
			Parameters: workflow.Parameters{
				"rand": workflow.ParameterSchema{
					Name:        "rand",
					Type:        "int",
					Validations: nil,
					Description: "Generated random number",
				},
			},
		},
	)
}

// RandFunction executes the rand function and returns a random number
func RandFunction(input *workflow.FunctionInput) (workflow.FunctionResult, error) {
	minValue := input.GetInt("min")
	maxValue := input.GetInt("max")

	randomNumberBig, err := rand.Int(rand.Reader, big.NewInt(int64(maxValue)))
	if err != nil {
		return nil, fmt.Errorf("failed to generate random number: %w", err)
	}

	randomNumber := int(randomNumberBig.Int64()) + minValue

	return workflow.NewFunctionResult(workflow.FunctionSuccess, map[string]any{
		"rand": randomNumber,
	}), nil
}
