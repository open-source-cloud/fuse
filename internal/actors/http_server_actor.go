package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/gofiber/fiber/v3"
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/server"
)

const HttpServerActor = "http_server_actor"

func NewHttpServerActor(cfg *config.Config) gen.ProcessBehavior {
	return &httpServerActor{
		config: cfg,
	}
}

type httpServerActor struct {
	act.Actor
	config *config.Config
	server *fiber.App
}

func (a *httpServerActor) Init(args ...any) error {
	// get the gen.Log interface using Log method of embedded gen.Process interface
	a.Log().Info("starting process %s", a.PID())

	a.server = server.New(a.config)

	return nil
}

func (a *httpServerActor) HandleMessage(from gen.PID, message any) error {
	a.Log().Info("got message from %s: %s", from, message)
	return nil
}

func (a *httpServerActor) Terminate(reason error) {
	a.Log().Info("%s terminated with reason: %s", a.PID(), reason)
}
