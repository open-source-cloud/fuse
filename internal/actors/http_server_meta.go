package actors

import (
	"ergo.services/ergo/gen"
	"github.com/gofiber/fiber/v3"
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/server"
)

const HttpServerMeta = "http_server_meta"

func NewHttpServerMeta(cfg *config.Config) gen.MetaBehavior {
	return &httpServerMeta{
		config:      cfg,
		messageChan: make(chan any, 10),
	}
}

type httpServerMeta struct {
	gen.MetaProcess
	config      *config.Config
	server      *fiber.App
	messageChan chan any
}

func (m *httpServerMeta) Init(meta gen.MetaProcess) error {
	m.MetaProcess = meta
	return nil
}

func (m *httpServerMeta) Start() error {
	m.Log().Info("starting '%s' process", HttpServerMeta)

	defer func() {
		err := m.server.Shutdown()
		if err != nil {
			m.Log().Error("Failed to shutdown server : %s", err)
			return
		}
	}()
	m.server = server.New(m.config, m.messageChan)

	for v := range m.messageChan {
		err := m.Send(m.Parent(), v)
		if err != nil {
			m.Log().Error("Failed to send message : %s", err)
		}
	}

	return nil
}

func (m *httpServerMeta) HandleMessage(from gen.PID, message any) error {
	return nil
}

func (m *httpServerMeta) HandleCall(from gen.PID, ref gen.Ref, request any) (any, error) {
	return nil, nil
}

func (m *httpServerMeta) Terminate(reason error) {}

func (m *httpServerMeta) HandleInspect(from gen.PID, item ...string) map[string]string {
	return nil
}
