package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
)

type WorkflowFuncFactory Factory[*WorkflowFunc]

func NewWorkflowFuncFactory() *WorkflowFuncFactory {
	return &WorkflowFuncFactory{
		Factory: func() gen.ProcessBehavior {
			return &WorkflowFunc{}
		},
	}
}

type WorkflowFunc struct {
	act.Actor
}

func (a *WorkflowFunc) Init(_ ...any) error {
	a.Log().Info("starting process %s", a.PID())

	return nil
}
