// Package di dependency injection
package di

import (
	"fmt"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/app"
	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/open-source-cloud/fuse/internal/readiness"
	"github.com/open-source-cloud/fuse/internal/logging"
	"github.com/open-source-cloud/fuse/internal/metrics"
	"github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/internal/services"
	"github.com/open-source-cloud/fuse/internal/tracing"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

func providePackageRegistration(pkgSvc services.PackageService) (app.PackagesReady, error) {
	if err := pkgSvc.RegisterInternalPackages(); err != nil {
		return app.PackagesReady{}, fmt.Errorf("failed to register internal packages: %w", err)
	}
	return app.PackagesReady{}, nil
}

// CommonModule FX module with base common providers
var CommonModule = fx.Module(
	"common",
	fx.Provide(
		// configs, loggers, cli
		logging.NewAppLogger,
		config.Instance,
		metrics.NewFuseMetrics,
		tracing.NewProvider,
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
		providePackageRegistration,
	),
)

// FuseAppModule FX module with the FUSE application providers
var FuseAppModule = fx.Module(
	"fuse_app",
	fx.Provide(
		readiness.NewFlag,
		app.NewApp,
	),
	fx.Invoke(func(_ gen.Node) {}),
)

// AllModules FX module with the complete application + base providers
var AllModules = fx.Options(
	CommonModule,
	PackageModule,
	DatabaseModule,
	ObjectStoreModule,
	IdempotencyModule,
	ConcurrencyModule,
	EventsModule,
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
