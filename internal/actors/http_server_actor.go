package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/gofiber/fiber/v3"
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/server"
	"github.com/rs/zerolog/log"
)

const HttpServerActor = "http_server_actor"

func NewHttpServerActor(cfg *config.Config) gen.ProcessBehavior {
	return &httpServerActor{
		config:      cfg,
		messageChan: make(chan any, 10),
	}
}

type httpServerActor struct {
	act.Actor
	config      *config.Config
	server      *fiber.App
	messageChan chan any
}

func (a *httpServerActor) Init(args ...any) error {
	// get the gen.Log interface using Log method of embedded gen.Process interface
	a.Log().Info("starting process %s", a.PID())

	a.server = server.New(a.config, a.messageChan)

	go func() {
		for v := range a.messageChan {
			err := a.Send(a.PID(), v)
			if err != nil {
				log.Error().Err(err).Msg("failed to send message")
			}
		}
	}()

	//err := a.Send(a.PID(), "init")
	//if err != nil {
	//	log.Error().Err(err).Msg("failed to send message")
	//	return err
	//}

	return nil
}

func (a *httpServerActor) HandleMessage(from gen.PID, message any) error {
	a.Log().Info("got message from %s: %s", from, message)

	err := a.Send(a.config.WorkflowPID, "message")
	if err != nil {
		log.Error().Err(err).Msg("failed to send message")
		return err
	}

	return nil
}

func (a *httpServerActor) Terminate(reason error) {
	a.Log().Info("%s terminated with reason: %s", a.PID(), reason)
}
