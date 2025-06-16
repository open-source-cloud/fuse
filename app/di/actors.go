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
		handlers.NewUpsertWorkflowSchemaHandlerFactory,
		handlers.NewTriggerWorkflowHandlerFactory,
		handlers.NewHealthCheckHandler,
		actors.NewWorkers,
	),
	fx.Invoke(func(
		workers *actors.Workers,
		healthCheckHandlerFactory *handlers.HealthCheckHandlerFactory,
		asyncFunctionResultHandlerFactory *handlers.AsyncFunctionResultHandlerFactory,
		upsertWorkflowSchemaHandlerFactory *handlers.UpsertWorkflowSchemaHandlerFactory,
		triggerWorkflowHandlerFactory *handlers.TriggerWorkflowHandlerFactory,
	) {
		workers.AddFactory(handlers.HealthCheckHandlerName, healthCheckHandlerFactory.Factory)
		workers.AddFactory(handlers.AsyncFunctionResultHandlerName, asyncFunctionResultHandlerFactory.Factory)
		workers.AddFactory(handlers.UpsertWorkflowSchemaHandlerName, upsertWorkflowSchemaHandlerFactory.Factory)
		workers.AddFactory(handlers.TriggerWorkflowHandlerName, triggerWorkflowHandlerFactory.Factory)
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
