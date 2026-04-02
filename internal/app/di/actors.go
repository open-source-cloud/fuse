package di

import (
	"github.com/open-source-cloud/fuse/internal/actors"
	"github.com/open-source-cloud/fuse/internal/handlers"
	"go.uber.org/fx"
)

// WorkerModule FX module with the worker providers
var WorkerModule = fx.Module(
	"worker",
	fx.Provide(
		handlers.NewAsyncFunctionResultHandlerFactory,
		handlers.NewWorkflowSchemaHandlerFactory,
		handlers.NewTriggerWorkflowHandlerFactory,
		handlers.NewHealthCheckHandler,
		handlers.NewPackagesHandler,
		handlers.NewRegisterPackageHandler,
		handlers.NewCancelWorkflowHandlerFactory,
		handlers.NewResolveAwakeableHandlerFactory,
		actors.NewWorkers,
	),
	fx.Invoke(func(
		workers *actors.Workers,
		healthCheckHandlerFactory *handlers.HealthCheckHandlerFactory,
		asyncFunctionResultHandlerFactory *handlers.AsyncFunctionResultHandlerFactory,
		workflowSchemaHandlerFactory *handlers.WorkflowSchemaHandlerFactory,
		triggerWorkflowHandlerFactory *handlers.TriggerWorkflowHandlerFactory,
		packagesHandlerFactory *handlers.PackagesHandlerFactory,
		registerPackageHandlerFactory *handlers.RegisterPackageHandlerFactory,
		cancelWorkflowHandlerFactory *handlers.CancelWorkflowHandlerFactory,
		resolveAwakeableHandlerFactory *handlers.ResolveAwakeableHandlerFactory,
	) {
		workers.AddFactory(handlers.HealthCheckHandlerName, healthCheckHandlerFactory.Factory)
		workers.AddFactory(handlers.AsyncFunctionResultHandlerName, asyncFunctionResultHandlerFactory.Factory)
		workers.AddFactory(handlers.WorkflowSchemaHandlerName, workflowSchemaHandlerFactory.Factory)
		workers.AddFactory(handlers.TriggerWorkflowHandlerName, triggerWorkflowHandlerFactory.Factory)
		workers.AddFactory(handlers.PackagesHandlerName, packagesHandlerFactory.Factory)
		workers.AddFactory(handlers.RegisterPackageHandlerName, registerPackageHandlerFactory.Factory)
		workers.AddFactory(handlers.CancelWorkflowHandlerName, cancelWorkflowHandlerFactory.Factory)
		workers.AddFactory(handlers.ResolveAwakeableHandlerName, resolveAwakeableHandlerFactory.Factory)
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
