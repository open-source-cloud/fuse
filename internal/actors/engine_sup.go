package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/workflow"
)

const EngineSupervisor = "engine_supervisor"

func NewEngineSupervisor(actorFactory *Factory, cfg *config.Config) gen.ProcessBehavior {
	return &engineSupervisor{
		actorFactory: actorFactory,
		config: cfg,
		engine: nil,
	}
}

type engineSupervisor struct {
	act.Supervisor
	actorFactory *Factory
	config *config.Config
	engine workflow.Engine
}

// Init invoked on a spawn Supervisor process. This is a mandatory callback for the implementation
func (sup *engineSupervisor) Init(_ ...any) (act.SupervisorSpec, error) {
	var spec act.SupervisorSpec

	// set supervisor type
	spec.Type = act.SupervisorTypeOneForOne

	// add children
	spec.Children = []act.SupervisorChildSpec{
		sup.actorFactory.SupervisorChildSpec(WorkflowSupervisor),
	}

	// set strategy
	spec.Restart.Strategy = act.SupervisorStrategyTransient
	spec.Restart.Intensity = 1 // How big bursts of restarts you want to tolerate.
	spec.Restart.Period = 5    // In seconds.

	return spec, nil
}

//
// Methods below are optional, so you can remove those that aren't being used
//

// HandleChildStart invoked on a successful child process starting if the option EnableHandleChild
// was enabled in act.SupervisorSpec
func (sup *engineSupervisor) HandleChildStart(name gen.Atom, pid gen.PID) error {
	return nil
}

// HandleChildTerminate invoked on a child process termination if the option EnableHandleChild
// was enabled in act.SupervisorSpec
func (sup *engineSupervisor) HandleChildTerminate(name gen.Atom, pid gen.PID, reason error) error {
	return nil
}

// HandleMessage invoked if Supervisor received a message sent with gen.Process.Send(...).
// Non-nil value of the returning error will cause termination of this process.
// To stop this process normally, return gen.TerminateReasonNormal or
// gen.TerminateReasonShutdown. Any other - for abnormal termination.
func (sup *engineSupervisor) HandleMessage(from gen.PID, message any) error {
	sup.Log().Info("supervisor got message from %s", from)
	return nil
}

// HandleCall invoked if Supervisor got a synchronous request made with gen.Process.Call(...).
// Return nil as a result to handle this request asynchronously and
// to provide the result later using the gen.Process.SendResponse(...) method.
func (sup *engineSupervisor) HandleCall(from gen.PID, ref gen.Ref, request any) (any, error) {
	sup.Log().Info("supervisor got request from %s with reference %s", from, ref)
	return gen.Atom("pong"), nil
}

// Terminate invoked on a termination supervisor process
func (sup *engineSupervisor) Terminate(reason error) {
	sup.Log().Info("supervisor terminated with reason: %s", reason)
}

// HandleInspect invoked on the request made with gen.Process.Inspect(...)
func (sup *engineSupervisor) HandleInspect(from gen.PID, item ...string) map[string]string {
	sup.Log().Info("supervisor got inspect request from %s", from)
	return nil
}
