package di

import (
	"github.com/open-source-cloud/fuse/internal/actors"
	"github.com/open-source-cloud/fuse/internal/handlers"
	"go.uber.org/fx"
)

type workerHandlerRegistrationParams struct {
	fx.In

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

// newWorkers builds the HTTP worker registry with all handler factories registered.
// It must be an fx.Provide (not Invoke) so *Workers is populated before gen.Node starts
// and MuxServerSup spawns pool children.
func newWorkers(p workerHandlerRegistrationParams) *actors.Workers {
	w := actors.NewWorkers()
	w.AddFactory(handlers.HealthCheckHandlerName, p.HealthCheckHandlerFactory.Factory)
	w.AddFactory(handlers.AsyncFunctionResultHandlerName, p.AsyncFunctionResultHandlerFactory.Factory)
	w.AddFactory(handlers.WorkflowSchemaHandlerName, p.WorkflowSchemaHandlerFactory.Factory)
	w.AddFactory(handlers.TriggerWorkflowHandlerName, p.TriggerWorkflowHandlerFactory.Factory)
	w.AddFactory(handlers.PackagesHandlerName, p.PackagesHandlerFactory.Factory)
	w.AddFactory(handlers.RegisterPackageHandlerName, p.RegisterPackageHandlerFactory.Factory)
	w.AddFactory(handlers.GetWorkflowHandlerName, p.GetWorkflowHandlerFactory.Factory)
	w.AddFactory(handlers.CancelWorkflowHandlerName, p.CancelWorkflowHandlerFactory.Factory)
	w.AddFactory(handlers.ResolveAwakeableHandlerName, p.ResolveAwakeableHandlerFactory.Factory)
	return w
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
		newWorkers,
	),
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
		actors.NewSchemaReplicationActorFactory,
	),
)
