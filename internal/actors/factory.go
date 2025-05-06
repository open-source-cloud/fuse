package actors

import (
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/rs/zerolog/log"
)

func NewFactory(cfg *config.Config) *Factory {
	return &Factory{
		config: cfg,
	}
}

type Factory struct {
	config *config.Config
}

func (f *Factory) Factory(actorName string) gen.ProcessFactory {
	switch actorName {
	case HttpServerActor:
		return func() gen.ProcessBehavior { return NewHttpServerActor(f.config) }
	case EngineSupervisor:
		return func() gen.ProcessBehavior { return NewEngineSupervisor(f.config) }

	default:
		log.Error().Msgf("unknown actor factory: %s", actorName)
		return nil
	}
}

func (f *Factory) ApplicationMemberSpecFactory(actorName string) gen.ApplicationMemberSpec {
	return gen.ApplicationMemberSpec{
		Name:    gen.Atom(actorName),
		Factory: f.Factory(actorName),
	}
}
