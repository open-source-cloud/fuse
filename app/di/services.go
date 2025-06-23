package di

import (
	"github.com/open-source-cloud/fuse/internal/services"
	"go.uber.org/fx"
)

var ServicesModule = fx.Module("services", fx.Provide(services.NewGraphService))
