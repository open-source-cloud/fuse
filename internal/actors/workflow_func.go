package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/packages"
	"github.com/rs/zerolog/log"
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
	a.Log().Info("starting process %s", a.PID())

	return nil
}

func (a *WorkflowFunc) HandleMessage(from gen.PID, message any) error {
	msg, ok := message.(messaging.Message)
	if !ok {
		a.Log().Error("message from %s is not a messaging.Message", from)
		return nil
	}
	a.Log().Info("got message from %s - %s", from, msg.Type)

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
		return err
	}
	fn, err := pkg.GetFunction(msgPayload.Function)
	if err != nil {
		a.Log().Error("Function %s is not registered in package %s", msgPayload.Function, msgPayload.PackageID)
		return err
	}
	result, err := fn.Execute(msgPayload.Params)
	a.Log().Info("execute function %s result: %s", fn.ID(), result)

	err = a.Send(WorkflowHandlerName(msgPayload.WorkflowID), nil)
	if err != nil {
		return err
	}

	return nil
}
