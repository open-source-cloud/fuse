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

func WorkflowHandlerName(workflowID workflow.ID) string {
	return fmt.Sprintf("workflow_handler_%s", workflowID.String())
}

type (
	WorkflowHandler struct {
		act.Actor

		config       *config.Config
		graphRepo    repos.GraphRepo
		workflowRepo repos.WorkflowRepo

		workflow *workflow.Workflow
	}

	WorkflowHandlerInitArgs struct {
		isNewWorkflow bool
		schemaID      string
		workflowID    workflow.ID
	}
)

func (a *WorkflowHandler) Init(args ...any) error {
	// get the gen.Log interface using Log method of embedded gen.Process interface
	a.Log().Info("starting process %s with args %s", a.PID(), args)

	if len(args) != 1 {
		return fmt.Errorf("workflow actor init args must be 1 == [WorkflowHandlerInitArgs]")
	}
	initArgs, ok := args[0].(WorkflowHandlerInitArgs)
	if !ok {
		return fmt.Errorf("workflow actor init args must be 1 == [workflow.ID]; got %T", args[0])
	}

	err := a.Send(a.PID(), messaging.NewActorInitMessage(initArgs))
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
		initArgs, ok := msg.Args.(WorkflowHandlerInitArgs)
		if !ok {
			a.Log().Error("failed to get workflowID from message: %s", msg)
			return fmt.Errorf("failed to get workflowID from message: %s", msg)
		}

		if initArgs.isNewWorkflow {
			graphRef, err := a.graphRepo.Get(initArgs.schemaID)
			if err != nil {
				a.Log().Error("failed to get graph for schema ID %s: %s", initArgs.schemaID, err)
				return gen.TerminateReasonPanic
			}
			a.workflow = workflow.New(initArgs.workflowID, graphRef)
			if a.workflowRepo.Save(a.workflow) != nil {
				a.Log().Error("failed to save workflow for ID %s: %s", initArgs.workflowID, err)
				return gen.TerminateReasonPanic
			}
			a.Log().Info("created new workflow with ID %s", initArgs.workflowID)
			a.workflow.Trigger()
		} else {
			var err error
			a.workflow, err = a.workflowRepo.Get(initArgs.workflowID.String())
			if err != nil {
				a.Log().Error("failed to get workflow for ID %s: %s", initArgs.workflowID, err)
				return gen.TerminateReasonPanic
			}
			a.Log().Info("got workflow with ID %s", initArgs.workflowID)
		}
	}

	return nil
}

func (a *WorkflowHandler) Terminate(reason error) {
	a.Log().Info("%s terminated with reason: %s", a.PID(), reason)
}
