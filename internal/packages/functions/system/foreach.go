package system

import (
	"github.com/open-source-cloud/fuse/internal/packages/transport"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// ForEachFunctionID foreach function ID
const ForEachFunctionID = "foreach"

// ForEachFullFunctionID is the full function ID for system/foreach
const ForEachFullFunctionID = PackageID + "/" + ForEachFunctionID

// ForEachFunctionMetadata returns the metadata of the foreach function.
// ForEach is a conditional-output node with two named output edges:
//   - "each": followed for every item/batch (spawned as a new iteration thread)
//   - "done": followed once after all items have been processed
//
// The conditional output field "_foreach_phase" is set to "done" when all
// iterations complete, so the graph engine routes to the correct edge.
func ForEachFunctionMetadata() workflow.FunctionMetadata {
	return workflow.FunctionMetadata{
		Transport: transport.Internal,
		Input: workflow.InputMetadata{
			Parameters: []workflow.ParameterSchema{
				{Name: "items", Type: "[]any", Required: true, Description: "Array of items to iterate over"},
				{Name: "batchSize", Type: "int", Required: false, Default: 1, Description: "Number of items per batch (1 = item-by-item)"},
				{Name: "concurrency", Type: "int", Required: false, Default: 1, Description: "Max concurrent batches running at once (1 = sequential)"},
			},
		},
		Output: workflow.OutputMetadata{
			ConditionalOutput:      true,
			ConditionalOutputField: "_foreach_phase",
			Parameters: []workflow.ParameterSchema{
				{Name: "item", Type: "any", Description: "Current item (when batchSize=1)"},
				{Name: "batch", Type: "[]any", Description: "Current batch (when batchSize>1)"},
				{Name: "index", Type: "int", Description: "Current index / batch number (0-based)"},
				{Name: "total", Type: "int", Description: "Total number of items"},
				{Name: "isLast", Type: "bool", Description: "True if this is the last item/batch"},
				{Name: "results", Type: "[]any", Description: "Aggregated results from all iterations (available on the done edge)"},
			},
			Edges: []workflow.OutputEdgeMetadata{
				{
					Name:  "each",
					Count: 1,
					ConditionalEdge: workflow.ConditionalEdgeMetadata{
						Value: "each",
					},
				},
				{
					Name:  "done",
					Count: 1,
					ConditionalEdge: workflow.ConditionalEdgeMetadata{
						Value: "done",
					},
				},
			},
		},
	}
}

// ForEachFunction is a placeholder — foreach is intercepted by the WorkflowHandler before execution.
// This function should never be called directly.
func ForEachFunction(_ *workflow.ExecutionInfo) (workflow.FunctionResult, error) {
	return workflow.NewFunctionResultSuccess(), nil
}
