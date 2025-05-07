package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/workflow"
)

const engineSupervisorName = "engine_supervisor"

func NewEngineSupervisorFactory(cfg *config.Config, workflowSupervisorFactory *Factory[*WorkflowSupervisor]) *Factory[*EngineSupervisor] {
	return &Factory[*EngineSupervisor]{
		Name: engineSupervisorName,
		Behavior: func() gen.ProcessBehavior {
			return &EngineSupervisor{
				config:                    cfg,
				workflowSupervisorFactory: workflowSupervisorFactory,
				engine:                    nil,
			}
		},
	}
}

type EngineSupervisor struct {
	act.Supervisor
	config                    *config.Config
	workflowSupervisorFactory *Factory[*WorkflowSupervisor]
	engine                    workflow.Engine
}

// Init invoked on a spawn Supervisor process. This is a mandatory callback for the implementation
func (s *EngineSupervisor) Init(_ ...any) (act.SupervisorSpec, error) {
	s.Log().Info("starting process %s", s.PID())

	var spec act.SupervisorSpec

	// set supervisor type
	spec.Type = act.SupervisorTypeOneForOne

	// add children
	spec.Children = []act.SupervisorChildSpec{
		s.workflowSupervisorFactory.SupervisorChildSpec(),
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
func (s *EngineSupervisor) HandleChildStart(name gen.Atom, pid gen.PID) error {
	return nil
}

// HandleChildTerminate invoked on a child process termination if the option EnableHandleChild
// was enabled in act.SupervisorSpec
func (s *EngineSupervisor) HandleChildTerminate(name gen.Atom, pid gen.PID, reason error) error {
	return nil
}

// HandleMessage invoked if Supervisor received a message sent with gen.Process.Send(...).
// Non-nil value of the returning error will cause termination of this process.
// To stop this process normally, return gen.TerminateReasonNormal or
// gen.TerminateReasonShutdown. Any other - for abnormal termination.
func (s *EngineSupervisor) HandleMessage(from gen.PID, message any) error {
	s.Log().Info("supervisor got message from %s", from)
	return nil
}

// HandleCall invoked if Supervisor got a synchronous request made with gen.Process.Call(...).
// Return nil as a result to handle this request asynchronously and
// to provide the result later using the gen.Process.SendResponse(...) method.
func (s *EngineSupervisor) HandleCall(from gen.PID, ref gen.Ref, request any) (any, error) {
	s.Log().Info("supervisor got request from %s with reference %s", from, ref)
	return gen.Atom("pong"), nil
}

// Terminate invoked on a termination supervisor process
func (s *EngineSupervisor) Terminate(reason error) {
	s.Log().Info("supervisor terminated with reason: %s", reason)
}

// HandleInspect invoked on the request made with gen.Process.Inspect(...)
func (s *EngineSupervisor) HandleInspect(from gen.PID, item ...string) map[string]string {
	s.Log().Info("supervisor got inspect request from %s", from)
	return nil
}
