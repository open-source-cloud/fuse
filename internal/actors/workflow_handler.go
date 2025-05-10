package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"fmt"
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/repos"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/pkg/uuid"
)

type WorkflowHandlerFactory Factory[*WorkflowHandler]

func NewWorkflowHandlerFactory(
	cfg *config.Config,
	graphRepo repos.GraphRepo,
	workflowRepo repos.WorkflowRepo,
) *WorkflowHandlerFactory {
	return &WorkflowHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &WorkflowHandler{
				config:       cfg,
				graphRepo:    graphRepo,
				workflowRepo: workflowRepo,
			}
		},
	}
}

type WorkflowHandler struct {
	act.Actor
	config       *config.Config
	graphRepo    repos.GraphRepo
	workflowRepo repos.WorkflowRepo
	workflow     *workflow.Workflow
}

func (a *WorkflowHandler) Init(args ...any) error {
	// get the gen.Log interface using Log method of embedded gen.Process interface
	a.Log().Info("starting process %s", a.PID())
	a.Log().Info("args: %s", args)

	if len(args) != 1 {
		return fmt.Errorf("workflow actor init args must be 1 == [workflowSchemaID/workflowID]")
	}
	workflowOrSchemaID, ok := args[0].(string)
	if !ok {
		return fmt.Errorf("workflow actor init args must be 1 == [workflowSchemaID/workflowID]")
	}

	err := a.Send(a.PID(), messaging.NewActorInitMessage(workflowOrSchemaID))
	if err != nil {
		return err
	}

	return nil
}

func (a *WorkflowHandler) HandleMessage(from gen.PID, message any) error {
	msg, ok := message.(messaging.Message)
	if !ok {
		a.Log().Error("message from %s is not a messaging.Message", from)
		return fmt.Errorf("message from %s is not a messaging.Message", from)
	}
	a.Log().Info("got message from %s - %s", from, msg.Type)

	switch msg.Type {
	case messaging.ActorInit:
		workflowOrSchemaID, ok := msg.Data.(string)
		if !ok {
			a.Log().Error("failed to get workflow or schema ID from message: %s", msg)
			return fmt.Errorf("failed to get workflow or schema ID from message: %s", msg)
		}

		if a.workflowRepo.Exists(workflowOrSchemaID) {
			a.workflow, _ = a.workflowRepo.Get(workflowOrSchemaID)
			a.Log().Info("%s workflow loaded", a.workflow.ID)
		} else {
			graph, err := a.graphRepo.Get(workflowOrSchemaID)
			if err != nil {
				a.Log().Error("failed to get graph for schema ID %s: %s", workflowOrSchemaID, err)
				return fmt.Errorf("failed to get graph for schema ID %s: %s", workflowOrSchemaID, err)
			}
			a.workflow = workflow.New(uuid.V7(), graph)
			err = a.workflowRepo.Save(a.workflow)
			if err != nil {
				a.Log().Error("failed to save workflow: %s", err)
				return fmt.Errorf("failed to save workflow: %s", err)
			}
			a.Log().Info("%s workflow created from schema ID %s", a.workflow.ID, workflowOrSchemaID)
		}

		err := a.Send(a.Parent(), messaging.NewChildInitMessage(a.workflow.ID))
		if err != nil {
			a.Log().Error("failed to send message to parent: %s", err)
			return err
		}
	}

	return nil
}

func (a *WorkflowHandler) Terminate(reason error) {
	a.Log().Info("%s terminated with reason: %s", a.PID(), reason)
}
