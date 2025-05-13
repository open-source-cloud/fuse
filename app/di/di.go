package di

import (
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/app"
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/actors"
	"github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/internal/packages/debug"
	"github.com/open-source-cloud/fuse/internal/packages/logic"
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
		// other services
		packages.NewPackageRegistry,
		// apps
		app.NewApp,
	),
	// eager loading
	fx.Invoke(func(
		_ repos.GraphRepo,
		_ repos.WorkflowRepo,
		registry packages.Registry,
		_ gen.Node,
	) {
		listOfInternalPackages := []packages.Package{
			debug.New(),
			logic.New(),
		}
		for _, pkg := range listOfInternalPackages {
			registry.Register(pkg.ID(), pkg)
		}
	}),
)
var AllModules = fx.Options(
	CommonModule,
	FuseAppModule,
	fx.WithLogger(logging.NewFxLogger()),
)

func Run(module ...fx.Option) {
	fx.New(module...).Run()
}
