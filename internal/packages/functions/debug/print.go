package debug

import (
	"errors"

	"github.com/open-source-cloud/fuse/pkg/utils"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/rs/zerolog/log"
)

const PrintFunctionID = "print"

func PrintFunctionMetadata() workflow.FunctionMetadata {
	return workflow.FunctionMetadata{
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
		},
	}
}

func PrintFunction(execInfo *workflow.ExecutionInfo) (workflow.FunctionResult, error) {
	message := execInfo.Input.GetStr("message")

	if message == "" {
		return workflow.NewFunctionResultError(errors.New("message is required"))
	}

	message = utils.ReplaceTokens(message, execInfo.Input.Raw())

	log.Info().Msgf("debug print: %s", message)

	return workflow.NewFunctionResultSuccess(), nil
}
