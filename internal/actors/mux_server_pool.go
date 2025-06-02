package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
)

const MuxServerPoolName = "mux_server_pool"

type MuxServerPoolFactory Factory[*muxServerPool]

func NewMuxServerPoolFactory(muxWebWorker *MuxWebWorkerFactory) *MuxServerPoolFactory {
	return &MuxServerPoolFactory{
		Factory: func() gen.ProcessBehavior {
			return &muxServerPool{
				muxWebWorker: muxWebWorker,
			}
		},
	}
}

type muxServerPool struct {
	act.Pool
	muxWebWorker *MuxWebWorkerFactory
}

// Init invoked on a spawn Pool for the initializing.
func (p *muxServerPool) Init(args ...any) (act.PoolOptions, error) {
	p.Log().Info("starting process pool of mux http server")

	opts := act.PoolOptions{
		WorkerFactory: p.muxWebWorker.Factory,
		PoolSize:      3,
	}

	p.Log().Info("started process pool of mux http server with %d workers", opts.PoolSize)

	return opts, nil
}
