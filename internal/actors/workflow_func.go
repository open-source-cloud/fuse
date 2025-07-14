package actors

import (
	"encoding/json"
	"github.com/open-source-cloud/fuse/internal/actors/actornames"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// WorkflowFuncFactory redefines a WorkflowFunc worker actor factory type for better readability
type WorkflowFuncFactory ActorFactory[*WorkflowFunc]

// NewWorkflowFuncFactory creates a dependency injection factory of WorkflowFunc worker actor
func NewWorkflowFuncFactory(packageRegistry packages.Registry) *WorkflowFuncFactory {
	return &WorkflowFuncFactory{
		Factory: func() gen.ProcessBehavior {
			return &WorkflowFunc{
				packageRegistry: packageRegistry,
			}
		},
	}
}

// WorkflowFunc defines a WorkflowFunc worker actor - used as a worker actor in WorkflowFuncPool actor for processing
//
//	workflow Functions execution
type WorkflowFunc struct {
	act.Actor

	packageRegistry packages.Registry
}

// Init called when a WorkflowFunc worker actor is being initialized
func (a *WorkflowFunc) Init(_ ...any) error {
	a.Log().Debug("starting process %s", a.PID())

	return nil
}

// HandleMessage handles messages sent to a WorkflowFunc worker actor
func (a *WorkflowFunc) HandleMessage(from gen.PID, message any) error {
	msg, ok := message.(messaging.Message)
	if !ok {
		a.Log().Error("message from %s is not a messaging.Message", from)
		return nil
	}
	a.Log().Info("got message from %s - %s", from, msg.Type)
	jsonArgs, _ := json.Marshal(msg.Args)
	a.Log().Debug("args: %s", string(jsonArgs))

	if msg.Type != messaging.ExecuteFunction {
		a.Log().Error("message from %s is not a messaging.ExecuteFunction; got %s", from, msg.Type)
		return nil
	}
	msgPayload, err := msg.ExecuteFunctionMessage()
	if err != nil {
		a.Log().Error("failed to get execute function message from message: %s", msg)
		return nil
	}
	pkg, err := a.packageRegistry.Get(msgPayload.PackageID)
	if err != nil {
		a.Log().Error("LoadedPackage %s is not registered", msgPayload.PackageID)
		return nil
	}
	input, err := workflow.NewFunctionInputWith(msgPayload.Input)
	if err != nil {
		a.Log().Error("failed to create function input: %s", err)
		return nil
	}

	result, err := pkg.ExecuteFunction(
		a,
		msgPayload.FunctionID,
		workflow.NewExecutionInfo(msgPayload.WorkflowID, msgPayload.ExecID, input),
	)
	if err != nil {
		a.Log().Error("failed to execute function %s: %s", msgPayload.FunctionID, err)
		return nil
	}
	jsonResult, _ := json.Marshal(result)
	a.Log().Debug("execute function %s result: %s", msgPayload.FunctionID, string(jsonResult))

	resultMsg := messaging.NewFunctionResultMessage(msgPayload.WorkflowID, msgPayload.ThreadID, msgPayload.ExecID, result)
	err = a.Send(actornames.WorkflowHandlerName(msgPayload.WorkflowID), resultMsg)
	if err != nil {
		return err
	}

	return nil
}
