package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
)

// MuxServerSupName is the name of the MuxServerSup actor
const MuxServerSupName = "mux_server_sup"

// MuxServerSupFactory is a factory for creating MuxServerSup actors
type MuxServerSupFactory ActorFactory[*MuxServerSup]

// NewMuxServerSupFactory creates a new MuxServerSupFactory
func NewMuxServerSupFactory(muxServer *MuxServerFactory) *MuxServerSupFactory {
	return &MuxServerSupFactory{
		Factory: func() gen.ProcessBehavior {
			return &MuxServerSup{
				muxServer: muxServer,
			}
		},
	}
}

// MuxServerSup is a supervisor actor for the MuxServer actor
type MuxServerSup struct {
	act.Supervisor
	muxServer *MuxServerFactory
}

// Init initializes the MuxServerSup actor
func (m *MuxServerSup) Init(args ...any) (act.SupervisorSpec, error) {
	m.Log().Info("starting mux server supervisor")

	spec := act.SupervisorSpec{
		Type: act.SupervisorTypeOneForOne,
		Children: []act.SupervisorChildSpec{
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
