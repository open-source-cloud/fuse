// Package di dependency injection
package di

import (
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/app"
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/actors"
	"github.com/open-source-cloud/fuse/internal/handlers"
	"github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/internal/packages/debug"
	"github.com/open-source-cloud/fuse/internal/packages/logic"
	"github.com/open-source-cloud/fuse/internal/repos"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/logging"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

// CommonModule FX module with base common providers
var CommonModule = fx.Module(
	"common",
	fx.Provide(
		// configs, loggers, cli
		logging.NewAppLogger,
		config.Instance,
	),
	fx.Invoke(func(_ zerolog.Logger) {
		// forces the initialization of the zerolog.Logger dependency
	}),
)

// WorkerModule FX module with the worker providers
var WorkerModule = fx.Module(
	"worker",
	fx.Provide(
		handlers.NewAsyncFunctionResultHandlerFactory,
		handlers.NewUpsertWorkflowSchemaHandlerFactory,
		handlers.NewTriggerWorkflowHandlerFactory,
		handlers.NewHealthCheckHandler,
		handlers.NewWorkers,
	),
	fx.Invoke(func(
		workers *handlers.Workers,
		healthCheckHandlerFactory *handlers.HealthCheckHandlerFactory,
		asyncFunctionResultHandlerFactory *handlers.AsyncFunctionResultHandlerFactory,
		upsertWorkflowSchemaHandlerFactory *handlers.UpsertWorkflowSchemaHandlerFactory,
		triggerWorkflowHandlerFactory *handlers.TriggerWorkflowHandlerFactory,
	) {
		workers.Factories.Add(handlers.HealthCheckHandlerName, healthCheckHandlerFactory.Factory)
		workers.Factories.Add(handlers.AsyncFunctionResultHandlerName, asyncFunctionResultHandlerFactory.Factory)
		workers.Factories.Add(handlers.UpsertWorkflowSchemaHandlerName, upsertWorkflowSchemaHandlerFactory.Factory)
		workers.Factories.Add(handlers.TriggerWorkflowHandlerName, triggerWorkflowHandlerFactory.Factory)
	}),
)

// ActorModule FX module with the actor providers
var ActorModule = fx.Module(
	"actor",
	fx.Provide(
		actors.NewMuxServerSupFactory,
		actors.NewMuxServerFactory,
		actors.NewWorkflowSupervisorFactory,
		actors.NewWorkflowInstanceSupervisorFactory,
		actors.NewWorkflowHandlerFactory,
		actors.NewWorkflowFuncPoolFactory,
		actors.NewWorkflowFuncFactory,
	),
)

// RepoModule FX module with the repo providers
var RepoModule = fx.Module(
	"repo",
	fx.Provide(
		repos.NewMemoryGraphRepo,
		repos.NewMemoryWorkflowRepo,
	),
)

// PackageModule FX module with the package providers
var PackageModule = fx.Module(
	"package",
	fx.Provide(
		packages.NewPackageRegistry,
	),
)

// WorkflowModule FX module with the workflow providers
var WorkflowModule = fx.Module(
	"workflow",
	fx.Provide(
		workflow.NewGraphFactory,
	),
)

// FuseAppModule FX module with the FUSE application providers
var FuseAppModule = fx.Module(
	"fuse_app",
	fx.Provide(
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

// AllModules FX module with the complete application + base providers
var AllModules = fx.Options(
	CommonModule,
	WorkerModule,
	ActorModule,
	RepoModule,
	PackageModule,
	WorkflowModule,
	FuseAppModule,
	fx.WithLogger(logging.NewFxLogger()),
)

// Run runs the FX dependency injection engine
func Run(module ...fx.Option) {
	fx.New(module...).Run()
}
