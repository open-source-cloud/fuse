package transport

import (
	"ergo.services/ergo/gen"
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
	// Snapshot handler PID now: WorkflowFunc may clear it after Execute returns, but async
	// callbacks (e.g. timer) run later and must not rely on reading the actor field again.
	handlerPID := gen.PID{}
	if hp, ok := handle.(actor.WorkflowHandlerPIDProvider); ok {
		handlerPID = hp.WorkflowHandlerPID()
	}
	execInfo.Finish = func(result workflow.FunctionOutput) {
		// Do not use handle.Send from a background goroutine: after HandleMessage returns,
		// the worker is in Sleep state and Process.Send returns gen.ErrNotAllowed.
		// Prefer captured PID from WorkflowFunc; Atom name routing can fail for pool workers.
		msg := messaging.NewAsyncFunctionResultMessage(execInfo.WorkflowID, execInfo.ExecID, result)
		to := any(gen.Atom(actornames.WorkflowHandlerName(execInfo.WorkflowID)))
		if handlerPID != (gen.PID{}) {
			to = handlerPID
		}
		err := handle.Node().Send(to, msg)
		if err != nil {
			log.Error().Err(err).Msg("failed to send async function result")
		}
	}
	return t.fn(execInfo)
}
