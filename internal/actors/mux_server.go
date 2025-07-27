package actors

import (
	"strconv"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"ergo.services/ergo/meta"
	"github.com/gorilla/mux"
	"github.com/open-source-cloud/fuse/internal/app/config"
)

// MuxServerFactory is a factory for creating MuxServer actors
type MuxServerFactory ActorFactory[*muxServer]

// muxServer is a mux server actor
type muxServer struct {
	act.Actor
	workers *Workers
	config  *config.Config
}

// NewMuxServerFactory creates a new MuxServerFactory
func NewMuxServerFactory(workers *Workers, config *config.Config) *MuxServerFactory {
	return &MuxServerFactory{
		Factory: func() gen.ProcessBehavior {
			return &muxServer{
				workers: workers,
				config:  config,
			}
		},
	}
}

// Init initializes the mux server
func (m *muxServer) Init(_ ...any) error {
	m.Log().Info("starting mux server")

	muxRouter := mux.NewRouter()

	// create routes
	for _, worker := range m.workers.GetAll() {
		if err := m.createWorkerPool(worker, muxRouter); err != nil {
			m.Log().Error("unable to create route for %s: %s", worker.Name, err)
			return err
		}
	}

	// create and spawn a web server meta-process
	// nolint:gosec // port is validated by the config
	port, err := strconv.Atoi(m.config.Server.Port)
	if err != nil {
		m.Log().Error("unable to convert port to int: %s", err)
		return err
	}

	// nolint:gosec // port is validated by the config
	serverOptions := meta.WebServerOptions{
		Port:    uint16(port),
		Host:    "localhost",
		Handler: muxRouter,
	}

	webserver, err := meta.CreateWebServer(serverOptions)
	if err != nil {
		m.Log().Error("unable to create Web server meta-process: %s", err)
		panic(err)
	}

	webServerID, err := m.SpawnMeta(webserver, gen.MetaOptions{})
	if err != nil {
		m.Log().Error("unable to spawn Web server meta-process: %s", err)
		panic(err)
	}

	httpProtocol := "http"
	m.Log().Info("started web server %s: use %s://%s:%d/", webServerID, httpProtocol, serverOptions.Host, serverOptions.Port)
	m.Log().Info("you may check it with command below:")
	m.Log().Info("$ curl -k %s://%s:%d/health", httpProtocol, serverOptions.Host, serverOptions.Port)

	return nil
}

// createWorkerPool creates a worker pool for a given route
func (m *muxServer) createWorkerPool(webWorker WebWorker, mux *mux.Router) error {
	workerPool := meta.CreateWebHandler(meta.WebHandlerOptions{
		Worker:         webWorker.PoolConfig.Name,
		RequestTimeout: webWorker.Timeout,
	})

	workerPoolID, err := m.SpawnMeta(workerPool, gen.MetaOptions{})
	if err != nil {
		m.Log().Error("unable to spawn WebHandler meta-process: %s", err)
		return err
	}

	m.Log().Info("started worker pool %s to serve %s (meta-process: %s)", webWorker.PoolConfig.Name, webWorker.Pattern, workerPoolID)

	mux.Handle(webWorker.Pattern, workerPool)

	return nil
}
