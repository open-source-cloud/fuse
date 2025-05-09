package app

import (
	"ergo.services/ergo/gen"
	"github.com/efectn/fx-zerolog"
	"github.com/open-source-cloud/fuse/app/cli"
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/actors"
	"github.com/open-source-cloud/fuse/internal/workflow"
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
			actors.NewWorkflowActorFactory,
			// workflow
			workflow.NewMemorySchemaRepo,
			// apps
			NewApp,
		),
		fx.Invoke(
			func(_ zerolog.Logger, _ *cli.Cli, app gen.Node) {},
		),
		fx.WithLogger(fxzerolog.Init()),
	).Run()
}
