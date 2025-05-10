package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"fmt"
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/repos"
	"github.com/open-source-cloud/fuse/internal/workflow"
)

type WorkflowHandlerFactory Factory[*WorkflowHandler]

func NewWorkflowHandlerFactory(
	cfg *config.Config,
	workflowRepo repos.WorkflowRepo,
) *WorkflowHandlerFactory {
	return &WorkflowHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &WorkflowHandler{
				config:       cfg,
				workflowRepo: workflowRepo,
			}
		},
	}
}

func WorkflowHandlerName(workflowID workflow.ID) string {
	return fmt.Sprintf("workflow_handler_%s", workflowID.String())
}

type WorkflowHandler struct {
	act.Actor
	config       *config.Config
	workflowRepo repos.WorkflowRepo
	workflow     *workflow.Workflow
}

func (a *WorkflowHandler) Init(args ...any) error {
	// get the gen.Log interface using Log method of embedded gen.Process interface
	a.Log().Info("starting process %s with args %s", a.PID(), args)

	if len(args) != 1 {
		return fmt.Errorf("workflow actor init args must be 1 == [workflow.ID]")
	}
	workflowID, ok := args[0].(workflow.ID)
	if !ok {
		return fmt.Errorf("workflow actor init args must be 1 == [workflow.ID]; got %T", args[0])
	}

	err := a.Send(a.PID(), messaging.NewActorInitMessage(workflowID))
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
		workflowID, ok := msg.Data.(workflow.ID)
		if !ok {
			a.Log().Error("failed to get workflowID from message: %s", msg)
			return fmt.Errorf("failed to get workflowID from message: %s", msg)
		}

		workflowRef, err := a.workflowRepo.Get(workflowID.String())
		if err != nil {
			a.Log().Error("failed to get workflow for ID %s: %s", workflowID, err)
			return err
		}
		a.workflow = workflowRef
	}

	return nil
}

func (a *WorkflowHandler) Terminate(reason error) {
	a.Log().Info("%s terminated with reason: %s", a.PID(), reason)
}
