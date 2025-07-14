package transport

import (
	"github.com/open-source-cloud/fuse/internal/actors/actor"
	"github.com/open-source-cloud/fuse/internal/actors/actornames"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/rs/zerolog/log"
)

// NewInternalFunctionTransport creates a new InternalFunctionTransport
func NewInternalFunctionTransport(fn workflow.Function) FunctionTransport {
	return &InternalFunctionTransport{
		fn: fn,
	}
}

// InternalFunctionTransport implements the Internal type of transport for calling Functions
type InternalFunctionTransport struct {
	fn workflow.Function
}

// Execute executes the function using internal transport
func (t *InternalFunctionTransport) Execute(handle actor.Handle, execInfo *workflow.ExecutionInfo) (workflow.FunctionResult, error) {
	execInfo.Finish = func(result workflow.FunctionOutput) {
		err := handle.Send(
			actornames.WorkflowHandlerName(execInfo.WorkflowID),
			messaging.NewAsyncFunctionResultMessage(execInfo.WorkflowID, execInfo.ExecID, result),
		)
		if err != nil {
			log.Error().Err(err).Msg("failed to send async function result")
		}
	}
	return t.fn(execInfo)
}
