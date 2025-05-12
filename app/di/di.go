package di

import (
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/app"
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/actors"
	"github.com/open-source-cloud/fuse/internal/repos"
	"github.com/open-source-cloud/fuse/logging"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

var CommonModule = fx.Module(
	"common",
	fx.Provide(
		// configs, loggers, cli
		logging.NewAppLogger,
		config.Instance,
	),
	fx.Invoke(func(_ zerolog.Logger) {}),
)
var FuseAppModule = fx.Module(
	"fuse_app",
	fx.Provide(
		// actors
		actors.NewHttpServerActorFactory,
		actors.NewWorkflowSupervisorFactory,
		actors.NewWorkflowInstanceSupervisorFactory,
		actors.NewWorkflowHandlerFactory,
		actors.NewWorkflowFuncPoolFactory,
		actors.NewWorkflowFuncFactory,
		// repositories
		repos.NewMemoryGraphRepo,
		repos.NewMemoryWorkflowRepo,
		// apps
		app.NewApp,
	),
	fx.Invoke(func(_ gen.Node) {}),
)
var AllModules = fx.Options(
	CommonModule,
	FuseAppModule,
	fx.WithLogger(logging.NewFxLogger()),
)

func Run(module ...fx.Option) {
	fx.New(module...).Run()
}
