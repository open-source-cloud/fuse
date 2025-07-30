package debug

import (
	"errors"

	"github.com/open-source-cloud/fuse/internal/packages/transport"
	"github.com/open-source-cloud/fuse/pkg/utils"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/rs/zerolog/log"
)

// PrintFunctionID is the id of the print function
const PrintFunctionID = "print"

// PrintFunctionMetadata returns the metadata for the print function
func PrintFunctionMetadata() workflow.FunctionMetadata {
	return workflow.FunctionMetadata{
		Transport: transport.Internal,
		Input: workflow.InputMetadata{
			CustomParameters: true,
			Parameters: []workflow.ParameterSchema{
				{
					Name: "message",
					Type: "string",
				},
			},
		},
		Output: workflow.OutputMetadata{
			Parameters: []workflow.ParameterSchema{},
			Edges:      make([]workflow.OutputEdgeMetadata, 0),
		},
	}
}

// PrintFunction prints a message to the console
func PrintFunction(execInfo *workflow.ExecutionInfo) (workflow.FunctionResult, error) {
	message := execInfo.Input.GetStr("message")

	if message == "" {
		return workflow.NewFunctionResultError(errors.New("message is required"))
	}

	message = utils.ReplaceTokens(message, execInfo.Input.Raw())

	log.Info().Msgf("debug print: %s", message)

	return workflow.NewFunctionResultSuccess(), nil
}
