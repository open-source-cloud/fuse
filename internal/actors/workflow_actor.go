package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/app/config"
)

const WorkflowActor = "workflow_actor"

func NewWorkflowActor(cfg *config.Config) gen.ProcessBehavior {
	return &workflowActor{
		config: cfg,
	}
}

type workflowActor struct {
	act.Actor
	config *config.Config
}

func (a *workflowActor) Init(args ...any) error {
	// get the gen.Log interface using Log method of embedded gen.Process interface
	a.Log().Info("starting process %s", a.PID())

	return nil
}

func (a *workflowActor) HandleMessage(from gen.PID, message any) error {
	a.Log().Info("got message from %s: %s", from, message)


	return nil
}

func (a *workflowActor) Terminate(reason error) {
	a.Log().Info("%s terminated with reason: %s", a.PID(), reason)
}
