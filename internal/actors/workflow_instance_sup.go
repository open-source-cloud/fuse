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
	workflowFuncPool *WorkflowFuncPoolFactory,
	workflowHandler *WorkflowHandlerFactory,
	workflowRepo repos.WorkflowRepo,
) *WorkflowInstanceSupervisorFactory {
	return &WorkflowInstanceSupervisorFactory{
		Factory: func() gen.ProcessBehavior {
			return &WorkflowInstanceSupervisor{
				config:          cfg,
				workflowFuncPool: workflowFuncPool,
				workflowHandler: workflowHandler,
				workflowRepo:    workflowRepo,
			}
		},
	}
}

type (
	WorkflowInstanceSupervisor struct {
		act.Supervisor

		config          *config.Config
		workflowFuncPool *WorkflowFuncPoolFactory
		workflowHandler *WorkflowHandlerFactory
		workflowRepo    repos.WorkflowRepo
	}
)

// Init invoked on a spawn Supervisor process. This is a mandatory callback for the implementation
func (a *WorkflowInstanceSupervisor) Init(args ...any) (act.SupervisorSpec, error) {
	a.Log().Info("starting process %s with args %s", a.PID(), args)

	if len(args) != 2 {
		return act.SupervisorSpec{}, fmt.Errorf("workflow instance supervisor init args must be 2 == [workflowID, workflowSchemaID]")
	}
	workflowID, ok := args[0].(workflow.ID)
	if !ok {
		return act.SupervisorSpec{}, fmt.Errorf("workflow instance supervisor init args must be 2 == [workflowID, workflowSchemaID]; first arg must be a workflow.ID, got %T", args[0])
	}
	schemaID, ok := args[1].(string)
	if !ok {
		return act.SupervisorSpec{}, fmt.Errorf("workflow instance supervisor init args must be 2 == [workflowID, workflowSchemaID]; second arg must be a string, got %T", args[1])
	}

	handlerInitArgs := WorkflowHandlerInitArgs{
		schemaID:      schemaID,
		workflowID:    workflowID,
	}

	// supervisor specification
	spec := act.SupervisorSpec{
		Type: act.SupervisorTypeOneForOne,
		// children
		Children: []act.SupervisorChildSpec{
			{
				Name:    gen.Atom(WorkflowFuncPoolName(workflowID)),
				Factory: a.workflowFuncPool.Factory,
				Args:    []any{},
			},
			{
				Name:        gen.Atom(WorkflowHandlerName(workflowID)),
				Factory:     a.workflowHandler.Factory,
				Args:        []any{handlerInitArgs},
			},
		},
		// strategy
		Restart: act.SupervisorRestart{
			Strategy:  act.SupervisorStrategyPermanent,
			Intensity: 2, // How big bursts of restarts you want to tolerate.
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
	a.Log().Debug("args: %s", msg.Args)

	return nil
}

// Terminate invoked on a termination process
func (a *WorkflowInstanceSupervisor) Terminate(reason error) {
	a.Log().Debug("process terminated with reason: %s", reason)
}

// HandleInspect invoked on the request made with gen.Process.Inspect(...)
func (a *WorkflowInstanceSupervisor) HandleInspect(from gen.PID, item ...string) map[string]string {
	a.Log().Debug("process got inspect request from %s", from)
	return nil
}

func (a *WorkflowInstanceSupervisor) HandleEvent(event gen.MessageEvent) error {
	a.Log().Debug("received event %s with value: %#v", event.Event, event.Message)
	return nil
}
