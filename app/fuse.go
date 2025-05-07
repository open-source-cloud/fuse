package app

import (
	"ergo.services/application/observer"
	"ergo.services/ergo"
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/actors"
	"github.com/rs/zerolog/log"
)

func NewApp(
	config *config.Config,
	engineSupervisorFactory *actors.Factory[*actors.EngineSupervisor],
	httpServerActorFactory *actors.Factory[*actors.HttpServerActor],
) (gen.Node, error) {
	var options gen.NodeOptions

	apps := make([]gen.ApplicationBehavior, 0, 2)
	if config.Params.ActorObserver {
		apps = append(apps, observer.CreateApp(observer.Options{}))
	}
	apps = append(apps, &Fuse{
		engineSupervisorFactory: engineSupervisorFactory,
		httpServerActorFactory:  httpServerActorFactory,
	})
	options.Applications = apps

	// disable default logger to get rid of multiple logging to the os.Stdout
	options.Log.DefaultLogger.Disable = true

	// add logger.
	logger, err := ErgoLogger()
	if err != nil {
		panic(err)
	}
	options.Log.Loggers = append(options.Log.Loggers, gen.Logger{Name: "zerolog", Logger: logger})

	node, err := ergo.StartNode("fuse@localhost", options)
	if err != nil {
		return nil, err
	}

	return node, nil
}

type Fuse struct {
	engineSupervisorFactory *actors.Factory[*actors.EngineSupervisor]
	httpServerActorFactory  *actors.Factory[*actors.HttpServerActor]
}

// Load invoked on loading application using the method ApplicationLoad of gen.Node interface.
func (app *Fuse) Load(_ gen.Node, _ ...any) (gen.ApplicationSpec, error) {
	return gen.ApplicationSpec{
		Name:        "fuse",
		Description: "description of this application",
		Mode:        gen.ApplicationModeTransient,
		Group: []gen.ApplicationMemberSpec{
			app.engineSupervisorFactory.ApplicationMemberSpec(),
			app.httpServerActorFactory.ApplicationMemberSpec(),
		},
	}, nil
}

// Start invoked once the application started
func (app *Fuse) Start(_ gen.ApplicationMode) {}

// Terminate invoked once the application stopped
func (app *Fuse) Terminate(_ error) {}

func (app *Fuse) Test() {
	log.Info().Msg("test")
}
