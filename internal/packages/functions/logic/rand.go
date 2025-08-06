package logic

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/open-source-cloud/fuse/internal/packages/transport"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// RandFunctionID rand function ID
const RandFunctionID = "rand"

// RandFunctionMetadata returns the metadata of the rand function
func RandFunctionMetadata() workflow.FunctionMetadata {
	return workflow.FunctionMetadata{
		Transport: transport.Internal,
		Input: workflow.InputMetadata{
			CustomParameters: false,
			Parameters: []workflow.ParameterSchema{
				{
					Name:        "min",
					Type:        "int",
					Required:    false,
					Validations: nil,
					Description: "Minimum value of the random number",
					Default:     1,
				},
				{
					Name:        "max",
					Type:        "int",
					Required:    false,
					Validations: nil,
					Description: "Maximum value of the random number",
					Default:     100,
				},
			},
			Edges: workflow.InputEdgeMetadata{
				Count:      0,
				Parameters: make([]workflow.ParameterSchema, 0),
			},
		},
		Output: workflow.OutputMetadata{
			Parameters: []workflow.ParameterSchema{
				{
					Name:        "rand",
					Type:        "int",
					Validations: nil,
					Description: "Generated random number",
				},
			},
			Edges: make([]workflow.OutputEdgeMetadata, 0),
		},
	}
}

// RandFunction executes the rand function and returns a random number
func RandFunction(execInfo *workflow.ExecutionInfo) (workflow.FunctionResult, error) {
	input := execInfo.Input

	minValue := input.GetInt("min")
	maxValue := input.GetInt("max")

	randomNumberBig, err := rand.Int(rand.Reader, big.NewInt(int64(maxValue)))
	if err != nil {
		return workflow.NewFunctionResultError(fmt.Errorf("failed to generate random number: %w", err))
	}

	randomNumber := int(randomNumberBig.Int64()) + minValue

	return workflow.NewFunctionResultSuccessWith(map[string]any{
		"rand": randomNumber,
	}), nil
}
