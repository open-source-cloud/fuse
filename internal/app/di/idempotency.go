package di

import (
	"context"

	"github.com/open-source-cloud/fuse/internal/idempotency"
	"go.uber.org/fx"
)

// IdempotencyModule FX module providing idempotency store
var IdempotencyModule = fx.Module(
	"idempotency",
	fx.Provide(provideIdempotencyStore),
)

func provideIdempotencyStore(lc fx.Lifecycle) idempotency.Store {
	ctx, cancel := context.WithCancel(context.Background())
	store := idempotency.NewMemoryStore(ctx)
	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			cancel()
			return nil
		},
	})
	return store
}
