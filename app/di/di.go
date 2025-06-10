// Package di dependency injection
package di

import (
	"context"
	"fmt"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/app"
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/actors"
	"github.com/open-source-cloud/fuse/internal/handlers"
	"github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/internal/packages/debug"
	"github.com/open-source-cloud/fuse/internal/packages/logic"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/logging"
	"github.com/rs/zerolog"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
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

// RepoModule FX module with the repo providers based on config
var RepoModule = fx.Module(
	"repo",
	fx.Provide(
		provideGraphRepository,
		provideWorkflowRepository,
		provideMongoClient,
	),
)

// provideGraphRepository provides the appropriate GraphRepository based on config
func provideGraphRepository(cfg *config.Config, mongoClient *mongo.Client) repositories.GraphRepository {
	switch cfg.Database.Driver {
	case "mongodb", "mongo":
		return repositories.NewMongoGraphRepository(mongoClient)
	case "memory", "":
		return repositories.NewMemoryGraphRepository()
	default:
		// Default to memory if unknown driver
		return repositories.NewMemoryGraphRepository()
	}
}

// provideWorkflowRepository provides the appropriate WorkflowRepository based on config
func provideWorkflowRepository(cfg *config.Config, mongoClient *mongo.Client) repositories.WorkflowRepository {
	switch cfg.Database.Driver {
	case "mongodb", "mongo":
		return repositories.NewMongoWorkflowRepository(mongoClient)
	case "memory", "":
		return repositories.NewMemoryWorkflowRepository()
	default:
		return repositories.NewMemoryWorkflowRepository()
	}
}

// provideMongoClient provides a MongoDB client if the driver is MongoDB, otherwise returns nil
func provideMongoClient(cfg *config.Config) *mongo.Client {
	if cfg.Database.Driver != "mongodb" && cfg.Database.Driver != "mongo" {
		return nil
	}

	// Build MongoDB connection string
	connectionString := fmt.Sprintf("mongodb://%s:%s/%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
	)

	if cfg.Database.User != "" && cfg.Database.Pass != "" {
		connectionString = fmt.Sprintf("mongodb://%s:%s@%s:%s/%s",
			cfg.Database.User,
			cfg.Database.Pass,
			cfg.Database.Host,
			cfg.Database.Port,
			cfg.Database.Name,
		)
	}

	clientOptions := options.Client().ApplyURI(connectionString)

	// Handle TLS configuration
	if cfg.Database.TLS {
		clientOptions = clientOptions.SetTLSConfig(nil) // Use default TLS config
	}

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to MongoDB: %v", err))
	}

	// Test the connection
	ctx := context.Background()
	if err := client.Ping(ctx, nil); err != nil {
		panic(fmt.Sprintf("Failed to ping MongoDB: %v", err))
	}

	return client
}

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
		_ repositories.GraphRepository,
		_ repositories.WorkflowRepository,
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
