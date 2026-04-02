package system

import (
	"github.com/open-source-cloud/fuse/internal/packages/transport"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// WaitFunctionID wait function ID
const WaitFunctionID = "wait"

// WaitFunctionMetadata returns the metadata of the wait function
func WaitFunctionMetadata() workflow.FunctionMetadata {
	return workflow.FunctionMetadata{
		Transport: transport.Internal,
		Input: workflow.InputMetadata{
			Parameters: []workflow.ParameterSchema{
				{Name: "timeout", Type: "string", Required: false, Description: "Max wait time (e.g., '30s', '24h'). 0 = no timeout"},
				{Name: "filter", Type: "string", Required: false, Description: "Optional expression to match incoming events"},
			},
		},
		Output: workflow.OutputMetadata{
			Parameters: []workflow.ParameterSchema{
				{Name: "data", Type: "map", Description: "Data from the resolving event"},
				{Name: "timedOut", Type: "bool", Description: "Whether the wait timed out"},
			},
		},
	}
}

// WaitFunction is a placeholder — wait is intercepted by the WorkflowHandler before execution.
// This function should never be called directly.
func WaitFunction(_ *workflow.ExecutionInfo) (workflow.FunctionResult, error) {
	return workflow.NewFunctionResultSuccess(), nil
}
