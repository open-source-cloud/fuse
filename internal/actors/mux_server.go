package actors

import (
	"net/http"
	"time"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"ergo.services/ergo/meta"
)

const MuxServerName = "mux_server"

type MuxServerFactory Factory[*muxServer]

func NewMuxServerFactory() *MuxServerFactory {
	return &MuxServerFactory{
		Factory: func() gen.ProcessBehavior {
			return &muxServer{}
		},
	}
}

type muxServer struct {
	act.Actor
}

func NewmuxServer() *muxServer {
	return &muxServer{}
}

func (m *muxServer) Init(args ...any) error {
	m.Log().Info("starting mux server")

	mux := http.NewServeMux()

	root := meta.CreateWebHandler(meta.WebHandlerOptions{
		Worker:         MuxServerPoolName,
		RequestTimeout: 10 * time.Second,
	})

	rootid, err := m.SpawnMeta(root, gen.MetaOptions{})
	if err != nil {
		m.Log().Error("unable to spawn WebHandler meta-process: %s", err)
		return err
	}

	mux.Handle("/", root)
	m.Log().Info("started WebHandler to serve '/' (meta-process: %s)", rootid)

	// create and spawn web server meta-process
	serverOptions := meta.WebServerOptions{
		Port:        9090,
		Host:        "localhost",
		CertManager: m.Node().CertManager(),
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
