package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/rs/zerolog/log"
)

const httpServerActorName = "http_server_actor"

func NewHttpServerActorFactory(cfg *config.Config) *Factory[*HttpServerActor] {
	return &Factory[*HttpServerActor]{
		Name: httpServerActorName,
		Behavior: func() gen.ProcessBehavior {
			return &HttpServerActor{
				config: cfg,
			}
		},
	}
}

type HttpServerActor struct {
	act.Actor
	config *config.Config
}

func (a *HttpServerActor) Init(args ...any) error {
	// get the gen.Log interface using Log method of embedded gen.Process interface
	a.Log().Info("starting process %s", a.PID())

	metaBehavior := NewHttpServerMeta(a.config)
	metaID, err := a.SpawnMeta(metaBehavior, gen.MetaOptions{})
	if err != nil {
		return err
	}
	a.Log().Info("meta '%s' spawned with metaID: %s", HttpServerMeta, metaID)

	return nil
}

func (a *HttpServerActor) HandleMessage(from gen.PID, message any) error {
	a.Log().Info("got message from %s:%s", from, message)

	err := a.Send(a.config.WorkflowPID, message)
	if err != nil {
		log.Error().Err(err).Msg("failed to send message")
		return err
	}

	return nil
}

func (a *HttpServerActor) Terminate(reason error) {
	a.Log().Info("%s terminated with reason: %s", a.PID(), reason)
}
