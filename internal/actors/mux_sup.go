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
func NewMuxServerSupFactory(muxServer *MuxServerFactory, workers *Workers) *MuxServerSupFactory {
	return &MuxServerSupFactory{
		Factory: func() gen.ProcessBehavior {
			return &MuxServerSup{
				muxServer: muxServer,
				workers:   workers,
			}
		},
	}
}

// MuxServerSup is a supervisor actor for the MuxServer actor
type MuxServerSup struct {
	act.Supervisor
	muxServer *MuxServerFactory
	workers   *Workers
}

// Init initializes the MuxServerSup actor
func (m *MuxServerSup) Init(_ ...any) (act.SupervisorSpec, error) {
	m.Log().Info("starting mux server supervisor")

	children := []act.SupervisorChildSpec{
		{
			Name:    MuxServerName,
			Factory: m.muxServer.Factory,
		},
	}

	for _, worker := range m.workers.GetAll() {
		workerFactory, ok := m.workers.GetFactory(string(worker.Name))
		if !ok {
			m.Log().Error("worker factory not found", "worker", worker.Name)
			continue
		}
		// Creates the worker pool dynamically based on the worker name and pool config
		pool := NewMuxWorkerPool(workerFactory, worker.PoolConfig)
		children = append(children, act.SupervisorChildSpec{
			Name:    worker.PoolConfig.Name,
			Factory: pool.Factory,
		})
		m.Log().Info("added worker pool", "worker", worker.Name, "pool", worker.PoolConfig.Name)
	}

	spec := act.SupervisorSpec{
		Type:     act.SupervisorTypeOneForOne,
		Children: children,
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
