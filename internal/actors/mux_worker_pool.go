package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
)

type MuxWorkerPoolFactory ActorFactory[*MuxWorkerPool]

type MuxWorkerPool struct {
	act.Pool
	workerFactory gen.ProcessFactory
	config        WorkerPoolConfig
}

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

func (w *MuxWorkerPool) Init(args ...any) (act.PoolOptions, error) {
	w.Log().Info("starting worker pool")

	opts := act.PoolOptions{
		WorkerFactory: w.workerFactory,
		PoolSize:      w.config.PoolSize,
	}

	return opts, nil
}
