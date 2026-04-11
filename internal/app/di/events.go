package di

import (
	"context"

	"github.com/open-source-cloud/fuse/internal/events"
	"go.uber.org/fx"
)

// EventsModule FX module providing the event bus
var EventsModule = fx.Module(
	"events",
	fx.Provide(provideEventBus),
)

func provideEventBus(lc fx.Lifecycle) events.EventBus {
	ctx, cancel := context.WithCancel(context.Background())
	bus := events.NewMemoryBus(ctx)
	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			cancel()
			return bus.Close()
		},
	})
	return bus
}
