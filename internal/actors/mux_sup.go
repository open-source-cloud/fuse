package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
)

const MuxServerSupName = "mux_server_sup"

type MuxServerSupFactory Factory[*MuxServerSup]

func NewMuxServerSupFactory(muxServerPool *MuxServerPoolFactory, muxServer *MuxServerFactory) *MuxServerSupFactory {
	return &MuxServerSupFactory{
		Factory: func() gen.ProcessBehavior {
			return &MuxServerSup{
				muxServerPool: muxServerPool,
				muxServer:     muxServer,
			}
		},
	}
}

type MuxServerSup struct {
	act.Supervisor
	muxServerPool *MuxServerPoolFactory
	muxServer     *MuxServerFactory
}

func (m *MuxServerSup) Init(args ...any) (act.SupervisorSpec, error) {
	m.Log().Info("starting mux server supervisor")

	spec := act.SupervisorSpec{
		Type: act.SupervisorTypeSimpleOneForOne,
		Children: []act.SupervisorChildSpec{
			{
				Name:    MuxServerPoolName,
				Factory: m.muxServerPool.Factory,
			},
			{
				Name:    MuxServerName,
				Factory: m.muxServer.Factory,
			},
		},
		Restart: act.SupervisorRestart{
			Strategy:  act.SupervisorStrategyTransient,
			Intensity: 2, // How big bursts of restarts you want to tolerate.
			Period:    5, // In seconds.
		},
		EnableHandleChild:   true,
		DisableAutoShutdown: true,
	}

	m.Log().Info("started mux server supervisor")

	return spec, nil
}

//
// Methods below are optional, so you can remove those that aren't be used
//

// HandleChildStart invoked on a successful child process starting if option EnableHandleChild
// was enabled in act.SupervisorSpec
func (sup *MuxServerSup) HandleChildStart(name gen.Atom, pid gen.PID) error {
	sup.Log().Info("supervisor got child start event for %s with pid %s", name, pid)
	return nil
}

// HandleChildTerminate invoked on a child process termination if option EnableHandleChild
// was enabled in act.SupervisorSpec
func (sup *MuxServerSup) HandleChildTerminate(name gen.Atom, pid gen.PID, reason error) error {
	sup.Log().Info("supervisor got child terminate event for %s with pid %s and reason %s", name, pid, reason)
	return nil
}

// HandleMessage invoked if Supervisor received a message sent with gen.Process.Send(...).
// Non-nil value of the returning error will cause termination of this process.
// To stop this process normally, return gen.TerminateReasonNormal or
// gen.TerminateReasonShutdown. Any other - for abnormal termination.
func (sup *MuxServerSup) HandleMessage(from gen.PID, message any) error {
	sup.Log().Info("supervisor got message from %s", from)
	return nil
}

// HandleCall invoked if Supervisor got a synchronous request made with gen.Process.Call(...).
// Return nil as a result to handle this request asynchronously and
// to provide the result later using the gen.Process.SendResponse(...) method.
func (sup *MuxServerSup) HandleCall(from gen.PID, ref gen.Ref, request any) (any, error) {
	sup.Log().Info("supervisor got request from %s with reference %s", from, ref)
	return gen.Atom("pong"), nil
}

// Terminate invoked on a termination supervisor process
func (sup *MuxServerSup) Terminate(reason error) {
	sup.Log().Info("supervisor terminated with reason: %s", reason)
}

// HandleInspect invoked on the request made with gen.Process.Inspect(...)
func (sup *MuxServerSup) HandleInspect(from gen.PID, item ...string) map[string]string {
	sup.Log().Info("supervisor got inspect request from %s", from)
	return nil
}
