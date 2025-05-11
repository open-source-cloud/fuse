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

type WorkflowInstanceSupervisorFactory Factory[*WorkflowInstanceSupervisor]

func NewWorkflowInstanceSupervisorFactory(
	cfg *config.Config,
	workflowHandler *WorkflowHandlerFactory,
	graphRepo repos.GraphRepo,
	workflowRepo repos.WorkflowRepo,
) *WorkflowInstanceSupervisorFactory {
	return &WorkflowInstanceSupervisorFactory{
		Factory: func() gen.ProcessBehavior {
			return &WorkflowInstanceSupervisor{
				config:          cfg,
				workflowHandler: workflowHandler,
				graphRepo:       graphRepo,
				workflowRepo:    workflowRepo,
			}
		},
	}
}

type (
	WorkflowInstanceSupervisor struct {
		act.Supervisor

		config          *config.Config
		workflowHandler *WorkflowHandlerFactory
		graphRepo       repos.GraphRepo
		workflowRepo    repos.WorkflowRepo

		workflow *workflow.Workflow
	}

	workflowInstanceActorInitArgs struct {
		isNewWorkflow bool
		schemaID      string
		workflowID    workflow.ID
	}
)

// Init invoked on a spawn Supervisor process. This is a mandatory callback for the implementation
func (a *WorkflowInstanceSupervisor) Init(args ...any) (act.SupervisorSpec, error) {
	a.Log().Info("starting process %s with args %s", a.PID(), args)

	if len(args) != 2 {
		return act.SupervisorSpec{}, fmt.Errorf("workflow instance supervisor init args must be 2 == [workflowSchemaID/workflowID, runNewWorkflow]")
	}
	workflowOrSchemaID, ok := args[0].(string)
	if !ok {
		return act.SupervisorSpec{}, fmt.Errorf("workflow instance supervisor init args must be 2 == [workflowSchemaID/workflowID, runNewWorkflow]; first arg must be a string, got %T", args[0])
	}
	runNewWorkflow, ok := args[1].(bool)
	if !ok {
		return act.SupervisorSpec{}, fmt.Errorf("workflow instance supervisor init args must be 2 == [workflowSchemaID/workflowID, runNewWorkflow]; second arg must be a bool, got %T", args[1])
	}

	var schemaID string
	var workflowID workflow.ID
	if runNewWorkflow {
		workflowID = workflow.NewID()
		schemaID = workflowOrSchemaID
	} else {
		workflowID = workflow.ID(workflowOrSchemaID)
	}

	err := a.Send(a.PID(), messaging.NewActorInitMessage(workflowInstanceActorInitArgs{
		isNewWorkflow: runNewWorkflow,
		schemaID:      schemaID,
		workflowID:    workflowID,
	}))
	if err != nil {
		a.Log().Error("failed to send message to workflow instance supervisor: %s", err)
		return act.SupervisorSpec{}, err
	}

	// supervisor specification
	spec := act.SupervisorSpec{
		Type: act.SupervisorTypeOneForOne,
		// children
		Children: []act.SupervisorChildSpec{
			{
				Name:    gen.Atom(WorkflowHandlerName(workflowID)),
				Factory: a.workflowHandler.Factory,
				Args:    []any{workflowID},
			},
		},
		// strategy
		Restart: act.SupervisorRestart{
			Strategy:  act.SupervisorStrategyPermanent,
			Intensity: 1, // How big bursts of restarts you want to tolerate.
			Period:    5, // In seconds.
		},
	}

	return spec, nil
}

// HandleMessage invoked if Supervisor received a message sent with gen.Process.Send(...).
// Non-nil value of the returning error will cause termination of this process.
// To stop this process normally, return gen.TerminateReasonNormal or
// gen.TerminateReasonShutdown. Any other - for abnormal termination.
func (a *WorkflowInstanceSupervisor) HandleMessage(from gen.PID, message any) error {
	msg, ok := message.(messaging.Message)
	if !ok {
		a.Log().Error("message from %s is not a messaging.Message", from)
		return fmt.Errorf("message from %s is not a messaging.Message", from)
	}
	a.Log().Info("got message from %s - %s", from, msg.Type)

	switch msg.Type {
	case messaging.ActorInit:
		args, ok := msg.Args.(workflowInstanceActorInitArgs)
		if !ok {
			a.Log().Error("failed to get workflowInstanceActorInitArgs from message: %s; got %T", msg.Type, msg.Args)
			return gen.TerminateReasonPanic
		}

		if args.isNewWorkflow {
			graphRef, err := a.graphRepo.Get(args.schemaID)
			if err != nil {
				a.Log().Error("failed to get graph for schema ID %s: %s", args.schemaID, err)
				return gen.TerminateReasonPanic
			}
			a.workflow = workflow.New(args.workflowID, graphRef)
			if a.workflowRepo.Save(a.workflow) != nil {
				a.Log().Error("failed to save workflow for ID %s: %s", args.workflowID, err)
				return gen.TerminateReasonPanic
			}
		} else {
			var err error
			a.workflow, err = a.workflowRepo.Get(args.workflowID.String())
			if err != nil {
				a.Log().Error("failed to get workflow for ID %s: %s", args.workflowID, err)
				return gen.TerminateReasonPanic
			}
		}
	}

	return nil
}

// Terminate invoked on a termination process
func (a *WorkflowInstanceSupervisor) Terminate(reason error) {
	a.Log().Info("process terminated with reason: %s", reason)
}

// HandleInspect invoked on the request made with gen.Process.Inspect(...)
func (a *WorkflowInstanceSupervisor) HandleInspect(from gen.PID, item ...string) map[string]string {
	a.Log().Info("process got inspect request from %s", from)
	return nil
}

func (a *WorkflowInstanceSupervisor) HandleEvent(event gen.MessageEvent) error {
	a.Log().Info("received event %s with value: %#v", event.Event, event.Message)
	return nil
}
