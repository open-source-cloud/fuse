package di

import (
	"github.com/open-source-cloud/fuse/internal/concurrency"
	"go.uber.org/fx"
)

// ConcurrencyModule FX module providing concurrency control primitives
var ConcurrencyModule = fx.Module(
	"concurrency",
	fx.Provide(
		concurrency.NewManager,
		concurrency.NewRateLimiter,
	),
)
