package transport

import (
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/actors/actor"
	"github.com/open-source-cloud/fuse/internal/actors/actornames"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/rs/zerolog/log"
)

// sendAsyncFunctionResult delivers async completion to the workflow handler.
//
// Ergo gen.Node.Send only accepts gen.Atom, gen.PID, gen.ProcessID, or gen.Alias; other types
// return gen.ErrUnsupported ("not supported"). From WorkflowFunc pool workers, sending with a
// captured workflow-handler gen.PID has been observed to fail that way; addressing the handler by
// its registered gen.Atom name on the local node matches sync Send(string, ...) and succeeds.
func sendAsyncFunctionResult(n gen.Node, wfID workflow.ID, execID workflow.ExecID, output workflow.FunctionOutput) error {
	if n == nil {
		return errNilNode
	}
	handlerName := gen.Atom(actornames.WorkflowHandlerName(wfID))
	msg := messaging.NewAsyncFunctionResultMessage(wfID, execID, output)
	return n.Send(handlerName, msg)
}

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
	if execInfo == nil {
		return workflow.FunctionResult{}, errNilExecutionInfo
	}
	execInfo.Finish = func(result workflow.FunctionOutput) {
		if execInfo == nil {
			log.Error().Msg("async Finish invoked with nil ExecutionInfo")
			return
		}
		n := handle.Node()
		if err := sendAsyncFunctionResult(n, execInfo.WorkflowID, execInfo.ExecID, result); err != nil {
			log.Error().Err(err).
				Str("workflowID", string(execInfo.WorkflowID)).
				Str("execID", execInfo.ExecID.String()).
				Msg("failed to send async function result")
		}
	}
	return t.fn(execInfo)
}
