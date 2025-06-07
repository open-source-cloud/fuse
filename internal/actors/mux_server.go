package actors

import (
	"net/http"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"ergo.services/ergo/meta"
	"github.com/open-source-cloud/fuse/internal/handlers"
)

// MuxServerName is the name of the MuxServer actor
const MuxServerName = "mux_server"

// MuxServerFactory is a factory for creating MuxServer actors
type MuxServerFactory ActorFactory[*muxServer]

// NewMuxServerFactory creates a new MuxServerFactory
func NewMuxServerFactory() *MuxServerFactory {
	return &MuxServerFactory{
		Factory: func() gen.ProcessBehavior {
			return &muxServer{}
		},
	}
}

// muxServer is a mux server actor
type muxServer struct {
	act.Actor
}

// NewMuxServer creates a new MuxServer actor
func NewMuxServer() *muxServer {
	return &muxServer{}
}

func (m *muxServer) Init(args ...any) error {
	m.Log().Info("starting mux server")

	mux := http.NewServeMux()

	// create routes
	workers := handlers.Workers()
	for _, worker := range workers {
		if err := m.createRoute(worker, mux); err != nil {
			m.Log().Error("unable to create route for %s: %s", worker.WorkerName, err)
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

func (m *muxServer) createRoute(route handlers.Worker, mux *http.ServeMux) error {
	webHandler := meta.CreateWebHandler(meta.WebHandlerOptions{
		Worker:         gen.Atom(route.WorkerName),
		RequestTimeout: route.Timeout,
	})

	workerID, err := m.SpawnMeta(webHandler, gen.MetaOptions{})
	if err != nil {
		m.Log().Error("unable to spawn WebHandler meta-process: %s", err)
		return err
	}

	m.Log().Info("started WebHandler to serve '%s' (meta-process: %s)", route.Pattern, workerID)

	mux.Handle(route.Pattern, webHandler)

	return nil
}
