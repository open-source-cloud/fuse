package system

import (
	"github.com/open-source-cloud/fuse/internal/packages/transport"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// SleepFunctionID sleep function ID
const SleepFunctionID = "sleep"

// SleepFunctionMetadata returns the metadata of the sleep function
func SleepFunctionMetadata() workflow.FunctionMetadata {
	return workflow.FunctionMetadata{
		Transport: transport.Internal,
		Input: workflow.InputMetadata{
			Parameters: []workflow.ParameterSchema{
				{Name: "duration", Type: "string", Required: true, Description: "Duration to sleep (e.g., '5s', '1h30m')"},
				{Name: "reason", Type: "string", Required: false, Description: "Human-readable reason for the delay"},
			},
		},
		Output: workflow.OutputMetadata{
			Parameters: []workflow.ParameterSchema{
				{Name: "sleptFor", Type: "string", Description: "Actual duration slept"},
			},
		},
	}
}

// SleepFunction is a placeholder — sleep is intercepted by the WorkflowHandler before execution.
// This function should never be called directly.
func SleepFunction(_ *workflow.ExecutionInfo) (workflow.FunctionResult, error) {
	return workflow.NewFunctionResultSuccess(), nil
}
