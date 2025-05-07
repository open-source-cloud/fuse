package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/app/config"
)

const workflowSupervisorName = "workflow_supervisor"

func NewWorkflowSupervisorFactory(cfg *config.Config, workflowActorFactory *Factory[*WorkflowActor]) *Factory[*WorkflowSupervisor] {
	return &Factory[*WorkflowSupervisor]{
		Name: workflowSupervisorName,
		Behavior: func() gen.ProcessBehavior {
			return &WorkflowSupervisor{
				config:               cfg,
				workflowActorFactory: workflowActorFactory,
			}
		},
	}
}

type WorkflowSupervisor struct {
	act.Supervisor
	config               *config.Config
	workflowActorFactory *Factory[*WorkflowActor]
}

// Init invoked on a spawn Supervisor process. This is a mandatory callback for the implementation
func (a *WorkflowSupervisor) Init(args ...any) (act.SupervisorSpec, error) {
	a.Log().Info("starting process %s", a.PID())
	var spec act.SupervisorSpec

	// set supervisor type
	spec.Type = act.SupervisorTypeSimpleOneForOne

	// add children
	spec.Children = []act.SupervisorChildSpec{
		a.workflowActorFactory.SupervisorChildSpec(),
	}

	// set strategy
	spec.DisableAutoShutdown = true
	spec.Restart.Strategy = act.SupervisorStrategyTransient
	spec.Restart.Intensity = 0 // How big bursts of restarts you want to tolerate.
	spec.Restart.Period = 5    // In seconds.

	a.config.WorkflowPID = a.PID()

	return spec, nil
}

//
// Methods below are optional, so you can remove those that aren't be used
//

// HandleMessage invoked if Pool received a message sent with gen.Process.Send(...) and
// with Priority higher than gen.MessagePriorityNormal. Any other messages are forwarded
// to the process from the pool.
// Non-nil value of the returning error will cause termination of this process.
// To stop this process normally, return gen.TerminateReasonNormal
// or any other for abnormal termination.
func (a *WorkflowSupervisor) HandleMessage(from gen.PID, message any) error {
	a.Log().Info("got message from %s:%s", from, message)

	return nil
}

// Terminate invoked on a termination process
func (a *WorkflowSupervisor) Terminate(reason error) {
	a.Log().Info("process terminated with reason: %s", reason)
}

// HandleInspect invoked on the request made with gen.Process.Inspect(...)
func (a *WorkflowSupervisor) HandleInspect(from gen.PID, item ...string) map[string]string {
	a.Log().Info("process got inspect request from %s", from)
	return nil
}

func (a *WorkflowSupervisor) HandleEvent(event gen.MessageEvent) error {
	a.Log().Info("received event %s with value: %#v", event.Event, event.Message)
	return nil
}
