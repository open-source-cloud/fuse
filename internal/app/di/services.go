package di

import (
	"github.com/open-source-cloud/fuse/internal/services"
	"go.uber.org/fx"
)

// ServicesModule provides the services for the application
var ServicesModule = fx.Module("services", fx.Provide(services.NewGraphService, services.NewPackageService))
