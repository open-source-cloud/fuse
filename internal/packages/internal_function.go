package packages

import (
	"fmt"
	"github.com/open-source-cloud/fuse/internal/actors/actor"
	"github.com/open-source-cloud/fuse/internal/actors/actornames"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/rs/zerolog/log"
)

type internalFunction struct {
	id       string
	metadata workflow.FunctionMetadata
	fn       workflow.Function
}

// NewInternalFunction creates a new internal function
func NewInternalFunction(packageID string, id string, metadata workflow.FunctionMetadata, fn workflow.Function) FunctionSpec {
	return &internalFunction{
		id:       fmt.Sprintf("%s/%s", packageID, id),
		metadata: metadata,
		fn:       fn,
	}
}

func (f *internalFunction) ID() string {
	return f.id
}

func (f *internalFunction) Metadata() workflow.FunctionMetadata {
	return f.metadata
}

func (f *internalFunction) Execute(handle actor.Handle, workflowID workflow.ID, execID workflow.ExecID, input *workflow.FunctionInput) (workflow.FunctionResult, error) {
	return f.fn(&workflow.ExecutionInfo{
		WorkflowID: workflowID,
		ExecID:     execID,
		Finish: func(result workflow.FunctionOutput) {
			err := handle.Send(
				actornames.WorkflowHandlerName(workflowID),
				messaging.NewAsyncFunctionResultMessage(workflowID, execID, result),
			)
			if err != nil {
				log.Error().Err(err).Msg("failed to send async function result")
			}
		},
	}, input)
}
