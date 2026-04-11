package di

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/open-source-cloud/fuse/internal/actors"
	"github.com/open-source-cloud/fuse/internal/handlers"
	"github.com/open-source-cloud/fuse/internal/repositories/postgres"
	"go.uber.org/fx"
)

type workerHandlerRegistrationParams struct {
	fx.In

	HealthCheckHandlerFactory           *handlers.HealthCheckHandlerFactory
	LivenessHandlerFactory              *handlers.LivenessHandlerFactory
	ReadinessHandlerFactory             *handlers.ReadinessHandlerFactory
	AsyncFunctionResultHandlerFactory   *handlers.AsyncFunctionResultHandlerFactory
	WorkflowSchemaHandlerFactory        *handlers.WorkflowSchemaHandlerFactory
	ListSchemasHandlerFactory           *handlers.ListSchemasHandlerFactory
	TriggerWorkflowHandlerFactory       *handlers.TriggerWorkflowHandlerFactory
	PackagesHandlerFactory              *handlers.PackagesHandlerFactory
	RegisterPackageHandlerFactory       *handlers.RegisterPackageHandlerFactory
	GetWorkflowHandlerFactory           *handlers.GetWorkflowHandlerFactory
	CancelWorkflowHandlerFactory        *handlers.CancelWorkflowHandlerFactory
	ResolveAwakeableHandlerFactory      *handlers.ResolveAwakeableHandlerFactory
	GetWorkflowSnapshotHandlerFactory   *handlers.GetWorkflowSnapshotHandlerFactory
	RetryNodeHandlerFactory             *handlers.RetryNodeHandlerFactory
	RetryWorkflowHandlerFactory         *handlers.RetryWorkflowHandlerFactory
	ListExecutionsHandlerFactory        *handlers.ListExecutionsHandlerFactory
	WorkflowTraceHandlerFactory         *handlers.WorkflowTraceHandlerFactory
	SchemaTracesHandlerFactory          *handlers.SchemaTracesHandlerFactory
	WebhookHandlerFactory               *handlers.WebhookHandlerFactory
	ListSchemaVersionsHandlerFactory    *handlers.ListSchemaVersionsHandlerFactory
	GetSchemaVersionHandlerFactory      *handlers.GetSchemaVersionHandlerFactory
	ActivateSchemaVersionHandlerFactory *handlers.ActivateSchemaVersionHandlerFactory
	RollbackSchemaHandlerFactory        *handlers.RollbackSchemaHandlerFactory
}

// newWorkers builds the HTTP worker registry with all handler factories registered.
// It must be an fx.Provide (not Invoke) so *Workers is populated before gen.Node starts
// and MuxServerSup spawns pool children.
func newWorkers(p workerHandlerRegistrationParams) *actors.Workers {
	w := actors.NewWorkers()
	w.AddFactory(handlers.HealthCheckHandlerName, p.HealthCheckHandlerFactory.Factory)
	w.AddFactory(handlers.LivenessHandlerName, p.LivenessHandlerFactory.Factory)
	w.AddFactory(handlers.ReadinessHandlerName, p.ReadinessHandlerFactory.Factory)
	w.AddFactory(handlers.AsyncFunctionResultHandlerName, p.AsyncFunctionResultHandlerFactory.Factory)
	w.AddFactory(handlers.WorkflowSchemaHandlerName, p.WorkflowSchemaHandlerFactory.Factory)
	w.AddFactory(handlers.ListSchemasHandlerName, p.ListSchemasHandlerFactory.Factory)
	w.AddFactory(handlers.TriggerWorkflowHandlerName, p.TriggerWorkflowHandlerFactory.Factory)
	w.AddFactory(handlers.PackagesHandlerName, p.PackagesHandlerFactory.Factory)
	w.AddFactory(handlers.RegisterPackageHandlerName, p.RegisterPackageHandlerFactory.Factory)
	w.AddFactory(handlers.GetWorkflowHandlerName, p.GetWorkflowHandlerFactory.Factory)
	w.AddFactory(handlers.CancelWorkflowHandlerName, p.CancelWorkflowHandlerFactory.Factory)
	w.AddFactory(handlers.ResolveAwakeableHandlerName, p.ResolveAwakeableHandlerFactory.Factory)
	w.AddFactory(handlers.GetWorkflowSnapshotHandlerName, p.GetWorkflowSnapshotHandlerFactory.Factory)
	w.AddFactory(handlers.RetryNodeHandlerName, p.RetryNodeHandlerFactory.Factory)
	w.AddFactory(handlers.RetryWorkflowHandlerName, p.RetryWorkflowHandlerFactory.Factory)
	w.AddFactory(handlers.ListExecutionsHandlerName, p.ListExecutionsHandlerFactory.Factory)
	w.AddFactory(handlers.WorkflowTraceHandlerName, p.WorkflowTraceHandlerFactory.Factory)
	w.AddFactory(handlers.SchemaTracesHandlerName, p.SchemaTracesHandlerFactory.Factory)
	w.AddFactory(handlers.WebhookHandlerName, p.WebhookHandlerFactory.Factory)
	w.AddFactory(handlers.ListSchemaVersionsHandlerName, p.ListSchemaVersionsHandlerFactory.Factory)
	w.AddFactory(handlers.GetSchemaVersionHandlerName, p.GetSchemaVersionHandlerFactory.Factory)
	w.AddFactory(handlers.ActivateSchemaVersionHandlerName, p.ActivateSchemaVersionHandlerFactory.Factory)
	w.AddFactory(handlers.RollbackSchemaHandlerName, p.RollbackSchemaHandlerFactory.Factory)
	return w
}

// WorkerModule FX module with the worker providers
var WorkerModule = fx.Module(
	"worker",
	fx.Provide(
		handlers.NewAsyncFunctionResultHandlerFactory,
		handlers.NewWorkflowSchemaHandlerFactory,
		handlers.NewListSchemasHandlerFactory,
		handlers.NewTriggerWorkflowHandlerFactory,
		handlers.NewHealthCheckHandler,
		handlers.NewLivenessHandler,
		provideReadinessHandlerFactory,
		handlers.NewPackagesHandler,
		handlers.NewRegisterPackageHandler,
		handlers.NewGetWorkflowHandlerFactory,
		handlers.NewCancelWorkflowHandlerFactory,
		handlers.NewResolveAwakeableHandlerFactory,
		handlers.NewGetWorkflowSnapshotHandlerFactory,
		handlers.NewRetryNodeHandlerFactory,
		handlers.NewRetryWorkflowHandlerFactory,
		handlers.NewListExecutionsHandlerFactory,
		handlers.NewWorkflowTraceHandlerFactory,
		handlers.NewSchemaTracesHandlerFactory,
		handlers.NewWebhookHandlerFactory,
		handlers.NewListSchemaVersionsHandlerFactory,
		handlers.NewGetSchemaVersionHandlerFactory,
		handlers.NewActivateSchemaVersionHandlerFactory,
		handlers.NewRollbackSchemaHandlerFactory,
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
		actors.NewWorkflowClaimActorFactory,
		actors.NewCronSchedulerFactory,
		actors.NewWebhookRouterFactory,
		actors.NewEventTriggerFactory,
		providePgListenerActorFactory,
	),
)

type pgListenerFactoryParams struct {
	fx.In
	Listener *postgres.PgListener `optional:"true"`
}

func providePgListenerActorFactory(p pgListenerFactoryParams) *actors.PgListenerActorFactory {
	if p.Listener == nil {
		return &actors.PgListenerActorFactory{}
	}
	return actors.NewPgListenerActorFactory(p.Listener)
}

type readinessHandlerParams struct {
	fx.In
	Pool *pgxpool.Pool `optional:"true"`
}

func provideReadinessHandlerFactory(p readinessHandlerParams) *handlers.ReadinessHandlerFactory {
	return handlers.NewReadinessHandlerFactory(p.Pool)
}
