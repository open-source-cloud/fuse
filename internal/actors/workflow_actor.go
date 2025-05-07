package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/app/config"
)

const workflowActorName = "workflow_actor"

func NewWorkflowActorFactory(cfg *config.Config) *Factory[*WorkflowActor] {
	return &Factory[*WorkflowActor]{
		Name: workflowActorName,
		Behavior: func() gen.ProcessBehavior {
			return &WorkflowActor{
				config: cfg,
			}
		},
	}
}

type WorkflowActor struct {
	act.Actor
	config *config.Config
}

func (a *WorkflowActor) Init(args ...any) error {
	// get the gen.Log interface using Log method of embedded gen.Process interface
	a.Log().Info("starting process %s", a.PID())

	return nil
}

func (a *WorkflowActor) HandleMessage(from gen.PID, message any) error {
	a.Log().Info("got message from %s: %s", from, message)

	return nil
}

func (a *WorkflowActor) Terminate(reason error) {
	a.Log().Info("%s terminated with reason: %s", a.PID(), reason)
}
