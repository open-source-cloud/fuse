package debug

import (
	"github.com/open-source-cloud/fuse/internal/packages/transport"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// NilFunctionID nil function ID
const NilFunctionID = "nil"

// NilFunctionMetadata nil function metadata
func NilFunctionMetadata() workflow.FunctionMetadata {
	return workflow.FunctionMetadata{
		Transport: transport.Internal,
		Input: workflow.InputMetadata{
			CustomParameters: false,
			Parameters:       make([]workflow.ParameterSchema, 0),
			Edges: workflow.InputEdgeMetadata{
				Count:      0,
				Parameters: make([]workflow.ParameterSchema, 0),
			},
		},
		Output: workflow.OutputMetadata{
			Parameters: make([]workflow.ParameterSchema, 0),
			Edges:      make([]workflow.OutputEdgeMetadata, 0),
		},
	}
}

// NilFunction executes the nil function
func NilFunction(_ *workflow.ExecutionInfo) (workflow.FunctionResult, error) {
	return workflow.NewFunctionResultSuccess(), nil
}
