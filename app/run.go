package app

import (
	"ergo.services/ergo/gen"
	"github.com/efectn/fx-zerolog"
	"github.com/open-source-cloud/fuse/app/cli"
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/actors"
	"github.com/open-source-cloud/fuse/internal/repos"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

func Run() {
	fx.New(
		fx.Provide(
			// configs, loggers, cli
			NewLogger,
			config.New,
			cli.New,
			// actors
			actors.NewHttpServerActorFactory,
			actors.NewWorkflowSupervisorFactory,
			actors.NewWorkflowInstanceSupervisorFactory,
			actors.NewWorkflowHandlerFactory,
			// repositories
			repos.NewMemoryGraphRepo,
			repos.NewMemoryWorkflowRepo,
			// apps
			NewApp,
		),
		fx.Invoke(
			func(_ zerolog.Logger, _ *cli.Cli, app gen.Node) {},
		),
		fx.WithLogger(fxzerolog.Init()),
	).Run()
}
