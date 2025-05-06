package actors

import (
	"ergo.services/ergo/act"
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
		return func() gen.ProcessBehavior { return NewEngineSupervisor(f, f.config) }
	case WorkflowSupervisor:
		return func() gen.ProcessBehavior { return NewWorkflowSupervisor(f, f.config) }
	case WorkflowActor:
		return func() gen.ProcessBehavior { return NewWorkflowActor(f.config) }

	default:
		log.Error().Msgf("unknown actor factory: %s", actorName)
		return nil
	}
}

func (f *Factory) ApplicationMemberSpec(actorName string) gen.ApplicationMemberSpec {
	return gen.ApplicationMemberSpec{
		Name:    gen.Atom(actorName),
		Factory: f.Factory(actorName),
	}
}

func (f *Factory) SupervisorChildSpec(actorName string) act.SupervisorChildSpec {
	return act.SupervisorChildSpec{
		Name:    gen.Atom(actorName),
		Factory: f.Factory(actorName),
	}
}
