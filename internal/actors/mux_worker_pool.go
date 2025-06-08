package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
)

// MuxWorkerPoolFactory is a factory for creating MuxWorkerPool actors
type MuxWorkerPoolFactory ActorFactory[*MuxWorkerPool]

// MuxWorkerPool is a pool of workers that handle HTTP requests
type MuxWorkerPool struct {
	act.Pool
	workerFactory gen.ProcessFactory
	config        WorkerPoolConfig
}

// NewMuxWorkerPool creates a new MuxWorkerPoolFactory
func NewMuxWorkerPool(workerFactory gen.ProcessFactory, config WorkerPoolConfig) *MuxWorkerPoolFactory {
	return &MuxWorkerPoolFactory{
		Factory: func() gen.ProcessBehavior {
			return &MuxWorkerPool{
				workerFactory: workerFactory,
				config:        config,
			}
		},
	}
}

// Init initializes the MuxWorkerPool
func (w *MuxWorkerPool) Init(_ ...any) (act.PoolOptions, error) {
	w.Log().Info("starting worker pool")

	opts := act.PoolOptions{
		WorkerFactory: w.workerFactory,
		PoolSize:      w.config.PoolSize,
	}

	return opts, nil
}
