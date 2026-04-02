package system

import (
	"github.com/open-source-cloud/fuse/internal/packages/transport"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// SubWorkflowFunctionID sub-workflow function ID
const SubWorkflowFunctionID = "subworkflow"

// SubWorkflowFullFunctionID is the full function ID for system/subworkflow
const SubWorkflowFullFunctionID = PackageID + "/" + SubWorkflowFunctionID

// SubWorkflowFunctionMetadata returns the metadata of the sub-workflow function
func SubWorkflowFunctionMetadata() workflow.FunctionMetadata {
	return workflow.FunctionMetadata{
		Transport: transport.Internal,
		Input: workflow.InputMetadata{
			Parameters: []workflow.ParameterSchema{
				{Name: "schemaId", Type: "string", Required: true, Description: "Schema ID of the workflow to execute"},
				{Name: "input", Type: "map", Required: false, Description: "Input data to pass to the sub-workflow trigger"},
				{Name: "async", Type: "bool", Required: false, Default: false, Description: "If true, don't wait for completion"},
			},
		},
		Output: workflow.OutputMetadata{
			Parameters: []workflow.ParameterSchema{
				{Name: "workflowId", Type: "string", Description: "ID of the spawned sub-workflow"},
				{Name: "status", Type: "string", Description: "Final status of the sub-workflow"},
				{Name: "output", Type: "map", Description: "Output data from the sub-workflow"},
			},
		},
	}
}

// SubWorkflowFunction is a placeholder — sub-workflow is intercepted by the WorkflowHandler.
func SubWorkflowFunction(_ *workflow.ExecutionInfo) (workflow.FunctionResult, error) {
	return workflow.NewFunctionResultSuccess(), nil
}
