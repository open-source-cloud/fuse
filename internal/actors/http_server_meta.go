package actors

import (
	"ergo.services/ergo/gen"
	"github.com/gofiber/fiber/v3"
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/repos"
	"github.com/open-source-cloud/fuse/internal/server"
	"github.com/open-source-cloud/fuse/internal/workflow"
)

// HTTPServerMeta actor name
const HTTPServerMeta = "http_server_meta"

// NewHTTPServerMeta dependency injection creation of HTTPServerMeta meta-process
func NewHTTPServerMeta(cfg *config.Config, graphFactory *workflow.GraphFactory, graphRepo repos.GraphRepo) gen.MetaBehavior {
	return &httpServerMeta{
		config:       cfg,
		graphFactory: graphFactory,
		graphRepo:    graphRepo,
		messageChan:  make(chan any, 10),
	}
}

type httpServerMeta struct {
	gen.MetaProcess
	config       *config.Config
	graphFactory *workflow.GraphFactory
	graphRepo    repos.GraphRepo
	server       *fiber.App
	messageChan  chan any
}

func (m *httpServerMeta) Init(meta gen.MetaProcess) error {
	m.MetaProcess = meta
	return nil
}

func (m *httpServerMeta) Start() error {
	m.Log().Debug("starting '%s' process", HTTPServerMeta)

	defer func() {
		err := m.server.Shutdown()
		if err != nil {
			m.Log().Error("Failed to shutdown server : %s", err)
			return
		}
	}()
	m.server = server.New(m.config, m.graphFactory, m.graphRepo, m.messageChan)

	for v := range m.messageChan {
		err := m.Send(m.Parent(), v)
		if err != nil {
			m.Log().Error("Failed to send message : %s", err)
		}
	}

	return nil
}

// HandleMessage (from gen.PID, message any) handles messages to HTTPServerMeta meta-process
func (m *httpServerMeta) HandleMessage(_ gen.PID, _ any) error {
	return nil
}

// HandleCall (from gen.PID, ref gen.Ref, request any) handles direct Calls to HTTPServerMeta meta-process
func (m *httpServerMeta) HandleCall(_ gen.PID, _ gen.Ref, _ any) (any, error) {
	return nil, nil
}

// Terminate called when HttpServerMeta meta-process gets terminated
func (m *httpServerMeta) Terminate(_ error) {}

// HandleInspect (from gen.PID, item ...string) called when HttpServerMeta meta-process gets inspected
func (m *httpServerMeta) HandleInspect(_ gen.PID, _ ...string) map[string]string {
	return nil
}
