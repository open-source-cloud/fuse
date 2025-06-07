package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"ergo.services/ergo/meta"
	"github.com/gorilla/mux"
	"github.com/open-source-cloud/fuse/internal/handlers"
)

// MuxServerName is the name of the MuxServer actor
const MuxServerName = "mux_server"

// MuxServerFactory is a factory for creating MuxServer actors
type MuxServerFactory ActorFactory[*muxServer]

// NewMuxServerFactory creates a new MuxServerFactory
func NewMuxServerFactory(workers *handlers.Workers) *MuxServerFactory {
	return &MuxServerFactory{
		Factory: func() gen.ProcessBehavior {
			return &muxServer{
				workers: workers,
			}
		},
	}
}

// muxServer is a mux server actor
type muxServer struct {
	act.Actor
	workers *handlers.Workers
}

// NewMuxServer creates a new MuxServer actor
func NewMuxServer() *muxServer {
	return &muxServer{}
}

func (m *muxServer) Init(args ...any) error {
	m.Log().Info("starting mux server")

	mux := mux.NewRouter()

	// create routes
	for _, worker := range m.workers.WebWorkers {
		if err := m.createWorkerPool(worker, mux); err != nil {
			m.Log().Error("unable to create route for %s: %s", worker.Name, err)
			return err
		}
	}

	// create and spawn web server meta-process
	serverOptions := meta.WebServerOptions{
		Port:        9090,
		Host:        "localhost",
		CertManager: nil,
		Handler:     mux,
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

	https := "http"
	if serverOptions.CertManager != nil {
		https = "https"
	}

	m.Log().Info("started Web server %s: use %s://%s:%d/", webserverid, https, serverOptions.Host, serverOptions.Port)
	m.Log().Info("you may check it with command below:")
	m.Log().Info("   $ curl -k %s://%s:%d", https, serverOptions.Host, serverOptions.Port)

	return nil
}

func (m *muxServer) createWorkerPool(route handlers.WebWorker, mux *mux.Router) error {
	workerPool := meta.CreateWebHandler(meta.WebHandlerOptions{
		Worker:         route.PoolConfig.Name,
		RequestTimeout: route.Timeout,
	})

	workerPoolID, err := m.SpawnMeta(workerPool, gen.MetaOptions{})
	if err != nil {
		m.Log().Error("unable to spawn WebHandler meta-process: %s", err)
		return err
	}

	m.Log().Info("started worker pool '%s' to serve '%s' (meta-process: %s)", route.PoolConfig.Name, route.Pattern, workerPoolID)

	mux.Handle(route.Pattern, workerPool)

	return nil
}
