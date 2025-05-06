package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/rs/zerolog/log"
)

type WorkflowSupervisor struct {
	act.Supervisor
}

func WorkflowSupervisorFactory() gen.ProcessBehavior {
	return &WorkflowSupervisor{}
}

// Init invoked on a spawn Supervisor process. This is a mandatory callback for the implementation
func (p *WorkflowSupervisor) Init(args ...any) (act.SupervisorSpec, error) {
	log.Info().Msg("WorkflowSupervisor:Init()")

	var spec act.SupervisorSpec

	// set supervisor type
	spec.Type = act.SupervisorTypeOneForOne

	// add children
	spec.Children = []act.SupervisorChildSpec{
		//{
		//	Name:    "workflow_worker",
		//	Factory: WorkflowWorkerFactory,
		//},
	}

	// set strategy
	spec.Restart.Strategy = act.SupervisorStrategyTransient
	spec.Restart.Intensity = 1 // How big bursts of restarts you want to tolerate.
	spec.Restart.Period = 5    // In seconds.

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
func (p *WorkflowSupervisor) HandleMessage(from gen.PID, message any) error {
	p.Log().Info("got message from %s", from)
	return nil
}

// HandleCall invoked if Pool got a synchronous request made with gen.Process.Call(...) and
// with Priority higher than gen.MessagePriorityNormal. Any other requests are forwarded
// to the process from the pool.
// Return nil as a result to handle this request asynchronously and
// to provide the result later using the gen.Process.SendResponse(...) method.
func (p *WorkflowSupervisor) HandleCall(from gen.PID, ref gen.Ref, request any) (any, error) {
	p.Log().Info("got request from %s with reference %s", from, ref)
	return gen.Atom("pong"), nil
}

// Terminate invoked on a termination process
func (p *WorkflowSupervisor) Terminate(reason error) {
	p.Log().Info("process terminated with reason: %s", reason)
}

// HandleInspect invoked on the request made with gen.Process.Inspect(...)
func (p *WorkflowSupervisor) HandleInspect(from gen.PID, item ...string) map[string]string {
	p.Log().Info("process got inspect request from %s", from)
	return nil
}
