package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"fmt"
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/repos"
)

const workflowHandlersSupervisorName = "workflow_handlers_sup"

func NewWorkflowHandlersSupervisorFactory(
	cfg *config.Config,
	graphRepo repos.GraphRepo,
	workflowActorFactory *Factory[*WorkflowHandler],
) *Factory[*WorkflowHandlersSupervisor] {
	return &Factory[*WorkflowHandlersSupervisor]{
		Name: workflowHandlersSupervisorName,
		Behavior: func() gen.ProcessBehavior {
			return &WorkflowHandlersSupervisor{
				config:                 cfg,
				workflowHandlerFactory: workflowActorFactory,
			}
		},
	}
}

type WorkflowHandlersSupervisor struct {
	act.Supervisor

	config                 *config.Config
	workflowHandlerFactory *Factory[*WorkflowHandler]
}

// Init invoked on a spawn Supervisor process. This is a mandatory callback for the implementation
func (a *WorkflowHandlersSupervisor) Init(_ ...any) (act.SupervisorSpec, error) {
	a.Log().Info("starting process %s", a.PID())

	// supervisor specification
	spec := act.SupervisorSpec{
		Type: act.SupervisorTypeOneForOne,
		// children
		Children: []act.SupervisorChildSpec{
			a.workflowHandlerFactory.SupervisorChildSpec(),
		},
		// strategy
		Restart: act.SupervisorRestart{
			Strategy:  act.SupervisorStrategyTransient,
			Intensity: 5, // How big bursts of restarts you want to tolerate.
			Period:    5, // In seconds.
		},
	}

	return spec, nil
}

// HandleMessage invoked if Supervisor received a message sent with gen.Process.Send(...).
// Non-nil value of the returning error will cause termination of this process.
// To stop this process normally, return gen.TerminateReasonNormal or
// gen.TerminateReasonShutdown. Any other - for abnormal termination.
func (a *WorkflowHandlersSupervisor) HandleMessage(from gen.PID, message any) error {
	msg, ok := message.(messaging.Message)
	if !ok {
		a.Log().Error("message from %s is not a messaging.Message", from)
		return fmt.Errorf("message from %s is not a messaging.Message", from)
	}
	a.Log().Info("got message from %s - %s", from, msg.Type)

	return nil
}

// Terminate invoked on a termination process
func (a *WorkflowHandlersSupervisor) Terminate(reason error) {
	a.Log().Info("process terminated with reason: %s", reason)
}

// HandleInspect invoked on the request made with gen.Process.Inspect(...)
func (a *WorkflowHandlersSupervisor) HandleInspect(from gen.PID, item ...string) map[string]string {
	a.Log().Info("process got inspect request from %s", from)
	return nil
}

func (a *WorkflowHandlersSupervisor) HandleEvent(event gen.MessageEvent) error {
	a.Log().Info("received event %s with value: %#v", event.Event, event.Message)
	return nil
}
