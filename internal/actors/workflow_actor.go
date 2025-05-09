package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"fmt"
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/repos"
)

const workflowActorName = "workflow"

func NewWorkflowActorFactory(cfg *config.Config, graphRepo repos.GraphRepo) *Factory[*WorkflowActor] {
	return &Factory[*WorkflowActor]{
		Name: workflowActorName,
		Behavior: func() gen.ProcessBehavior {
			return &WorkflowActor{
				config:    cfg,
				graphRepo: graphRepo,
			}
		},
	}
}

type WorkflowActor struct {
	act.Actor
	config           *config.Config
	graphRepo        repos.GraphRepo
	workflowID       string
	workflowSchemaID string
}

func (a *WorkflowActor) Init(args ...any) error {
	// get the gen.Log interface using Log method of embedded gen.Process interface
	a.Log().Info("starting process %s", a.PID())
	a.Log().Info("args: %s", args)

	if len(args) != 1 {
		return fmt.Errorf("workflow actor init args must be 1 == [workflowSchemaID]")
	}
	schemaID, ok := args[0].(string)
	if !ok {
		return fmt.Errorf("workflow actor init args must be 1 == [workflowSchemaID]")
	}
	a.workflowSchemaID = schemaID

	err := a.Send(a.PID(), messaging.NewActorInitMessage())
	if err != nil {
		return err
	}

	return nil
}

func (a *WorkflowActor) HandleMessage(from gen.PID, message any) error {
	msg, ok := message.(messaging.Message)
	if !ok {
		a.Log().Error("message from %s is not a messaging.Message", from)
		return fmt.Errorf("message from %s is not a messaging.Message", from)
	}
	a.Log().Info("got message from %s - %s", from, msg.Type)

	switch msg.Type {
	case messaging.ActorInit:
		err := a.Send(a.Parent(), messaging.NewChildInitMessage(a.workflowSchemaID))
		if err != nil {
			a.Log().Error("failed to send message to parent: %s", err)
			return err
		}
	}

	return nil
}

func (a *WorkflowActor) Terminate(reason error) {
	a.Log().Info("%s terminated with reason: %s", a.PID(), reason)
}
