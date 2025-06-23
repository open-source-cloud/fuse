// Package di dependency injection
package di

import (
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/app"
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/internal/packages/debug"
	"github.com/open-source-cloud/fuse/internal/packages/logic"
	"github.com/open-source-cloud/fuse/internal/repositories"
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

// PackageModule FX module with the package providers
var PackageModule = fx.Module(
	"package",
	fx.Provide(
		packages.NewPackageRegistry,
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
	MongoModule,
	RepoModule,
	ServicesModule,
	WorkerModule,
	ActorModule,
	PackageModule,
	FuseAppModule,
	fx.WithLogger(logging.NewFxLogger()),
)

// Run runs the FX dependency injection engine
func Run(module ...fx.Option) {
	fx.New(module...).Run()
}
