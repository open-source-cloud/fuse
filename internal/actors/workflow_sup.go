package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/app/config"
)

const WorkflowSupervisor = "workflow_supervisor"

type workflowSupervisor struct {
	act.Supervisor
	actorFactory *Factory
	config       *config.Config
}

func NewWorkflowSupervisor(actorFactory *Factory, cfg *config.Config) gen.ProcessBehavior {
	return &workflowSupervisor{
		actorFactory: actorFactory,
		config:       cfg,
	}
}

// Init invoked on a spawn Supervisor process. This is a mandatory callback for the implementation
func (a *workflowSupervisor) Init(args ...any) (act.SupervisorSpec, error) {
	var spec act.SupervisorSpec

	// set supervisor type
	spec.Type = act.SupervisorTypeOneForOne

	// add children
	spec.Children = []act.SupervisorChildSpec{
		a.actorFactory.SupervisorChildSpec(WorkflowActor),
	}

	// set strategy
	spec.Restart.Strategy = act.SupervisorStrategyTransient
	spec.Restart.Intensity = 1 // How big bursts of restarts you want to tolerate.
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
func (a *workflowSupervisor) HandleMessage(from gen.PID, message any) error {
	a.Log().Info("got message from %s:%s", from, message)

	return nil
}

// Terminate invoked on a termination process
func (a *workflowSupervisor) Terminate(reason error) {
	a.Log().Info("process terminated with reason: %s", reason)
}

// HandleInspect invoked on the request made with gen.Process.Inspect(...)
func (a *workflowSupervisor) HandleInspect(from gen.PID, item ...string) map[string]string {
	a.Log().Info("process got inspect request from %s", from)
	return nil
}

func (a *workflowSupervisor) HandleEvent(event gen.MessageEvent) error {
	a.Log().Info("received event %s with value: %#v", event.Event, event.Message)
	return nil
}
