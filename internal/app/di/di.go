// Package di dependency injection
package di

import (
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/app"
	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/open-source-cloud/fuse/internal/logging"
	"github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/internal/services"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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
		packages.NewInternal,
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
		_ packages.Registry,
		_ gen.Node,
		pkgSvc services.PackageService,
	) {
		// register internal packages
		if err := pkgSvc.RegisterInternalPackages(); err != nil {
			log.Error().Err(err).Msg("failed to register internal packages")
			panic(err)
		}
	}),
)

// AllModules FX module with the complete application + base providers
var AllModules = fx.Options(
	CommonModule,
	PackageModule,
	MongoModule,
	RepoModule,
	ServicesModule,
	WorkerModule,
	ActorModule,
	FuseAppModule,
	fx.WithLogger(logging.NewFxLogger()),
)

// Run runs the FX dependency injection engine
func Run(module ...fx.Option) {
	fx.New(module...).Run()
}
