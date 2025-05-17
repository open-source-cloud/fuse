package actors

import (
	"encoding/json"
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

type WorkflowFuncFactory Factory[*WorkflowFunc]

func NewWorkflowFuncFactory(packageRegistry packages.Registry) *WorkflowFuncFactory {
	return &WorkflowFuncFactory{
		Factory: func() gen.ProcessBehavior {
			return &WorkflowFunc{
				packageRegistry: packageRegistry,
			}
		},
	}
}

type WorkflowFunc struct {
	act.Actor

	packageRegistry packages.Registry
}

func (a *WorkflowFunc) Init(_ ...any) error {
	a.Log().Debug("starting process %s", a.PID())

	return nil
}

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
		a.Log().Error("Package %s is not registered", msgPayload.PackageID)
		return nil
	}
	fn, err := pkg.GetFunction(msgPayload.FunctionID)
	if err != nil {
		a.Log().Error("FunctionID %s is not registered in package %s", msgPayload.FunctionID, msgPayload.PackageID)
		return nil
	}

	input, err := workflow.NewFunctionInputWith(msgPayload.Input)
	if err != nil {
		a.Log().Error("failed to create function input: %s", err)
		return nil
	}
	result, err := fn.Execute(input)
	if err != nil {
		a.Log().Error("failed to execute function %s: %s", fn.ID(), err)
		return nil
	}
	a.Log().Debug("execute function %s result: %s", fn.ID(), result)

	resultMsg := messaging.NewFunctionResultMessage(msgPayload.WorkflowID, msgPayload.ThreadID, msgPayload.ExecID, result)
	err = a.Send(WorkflowHandlerName(msgPayload.WorkflowID), resultMsg)
	if err != nil {
		return err
	}

	return nil
}
