package di

import (
	"github.com/open-source-cloud/fuse/internal/actors"
	"github.com/open-source-cloud/fuse/internal/handlers"
	"go.uber.org/fx"
)

type workerHandlerRegistrationParams struct {
	fx.In

	Workers                           *actors.Workers
	HealthCheckHandlerFactory         *handlers.HealthCheckHandlerFactory
	AsyncFunctionResultHandlerFactory *handlers.AsyncFunctionResultHandlerFactory
	WorkflowSchemaHandlerFactory      *handlers.WorkflowSchemaHandlerFactory
	TriggerWorkflowHandlerFactory     *handlers.TriggerWorkflowHandlerFactory
	PackagesHandlerFactory            *handlers.PackagesHandlerFactory
	RegisterPackageHandlerFactory     *handlers.RegisterPackageHandlerFactory
	GetWorkflowHandlerFactory         *handlers.GetWorkflowHandlerFactory
	CancelWorkflowHandlerFactory      *handlers.CancelWorkflowHandlerFactory
	ResolveAwakeableHandlerFactory    *handlers.ResolveAwakeableHandlerFactory
}

func registerWorkerHandlers(p workerHandlerRegistrationParams) {
	p.Workers.AddFactory(handlers.HealthCheckHandlerName, p.HealthCheckHandlerFactory.Factory)
	p.Workers.AddFactory(handlers.AsyncFunctionResultHandlerName, p.AsyncFunctionResultHandlerFactory.Factory)
	p.Workers.AddFactory(handlers.WorkflowSchemaHandlerName, p.WorkflowSchemaHandlerFactory.Factory)
	p.Workers.AddFactory(handlers.TriggerWorkflowHandlerName, p.TriggerWorkflowHandlerFactory.Factory)
	p.Workers.AddFactory(handlers.PackagesHandlerName, p.PackagesHandlerFactory.Factory)
	p.Workers.AddFactory(handlers.RegisterPackageHandlerName, p.RegisterPackageHandlerFactory.Factory)
	p.Workers.AddFactory(handlers.GetWorkflowHandlerName, p.GetWorkflowHandlerFactory.Factory)
	p.Workers.AddFactory(handlers.CancelWorkflowHandlerName, p.CancelWorkflowHandlerFactory.Factory)
	p.Workers.AddFactory(handlers.ResolveAwakeableHandlerName, p.ResolveAwakeableHandlerFactory.Factory)
}

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
		handlers.NewGetWorkflowHandlerFactory,
		handlers.NewCancelWorkflowHandlerFactory,
		handlers.NewResolveAwakeableHandlerFactory,
		actors.NewWorkers,
	),
	fx.Invoke(registerWorkerHandlers),
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
