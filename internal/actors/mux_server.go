package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"ergo.services/ergo/meta"
	"github.com/gorilla/mux"
	"github.com/open-source-cloud/fuse/app/config"
)

// MuxServerName is the name of the MuxServer actor
const MuxServerName = "mux_server"

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

	mux := mux.NewRouter()

	// create routes
	for _, worker := range m.workers.GetAll() {
		if err := m.createWorkerPool(worker, mux); err != nil {
			m.Log().Error("unable to create route for %s: %s", worker.Name, err)
			return err
		}
	}

	// create and spawn web server meta-process
	serverOptions := meta.WebServerOptions{
		Port:    m.config.Server.Port,
		Host:    m.config.Server.Host,
		Handler: mux,
	}

	webserver, err := meta.CreateWebServer(serverOptions)
	if err != nil {
		m.Log().Error("unable to create Web server meta-process: %s", err)
		return err
	}

	webserverid, err := m.SpawnMeta(webserver, gen.MetaOptions{})
	if err != nil {
		m.Log().Error("unable to spawn Web server meta-process: %s", err)
		return err
	}

	m.Log().Info("started web server %s: use http://%s:%d/", webserverid, serverOptions.Host, serverOptions.Port)
	m.Log().Info("you may check it with command below:")
	m.Log().Info("$ curl -k http://%s:%d/health", serverOptions.Host, serverOptions.Port)

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

	m.Log().Info("started worker pool '%s' to serve '%s' (meta-process: %s)", webWorker.PoolConfig.Name, webWorker.Pattern, workerPoolID)

	mux.Handle(webWorker.Pattern, workerPool)

	return nil
}
