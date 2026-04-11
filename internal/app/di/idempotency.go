package di

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/open-source-cloud/fuse/internal/idempotency"
	"github.com/open-source-cloud/fuse/internal/repositories/postgres"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
)

// IdempotencyModule FX module providing idempotency store
var IdempotencyModule = fx.Module(
	"idempotency",
	fx.Provide(provideIdempotencyStore),
)

type idempotencyParams struct {
	fx.In
	Lifecycle fx.Lifecycle
	Config    *config.Config
	Pool      *pgxpool.Pool `optional:"true"`
}

func provideIdempotencyStore(p idempotencyParams) idempotency.Store {
	if p.Config.Database.Driver == config.DBDriverPostgres && p.Pool != nil {
		log.Debug().Msg("using postgres idempotency store (HA-safe)")
		return postgres.NewIdempotencyStore(p.Pool)
	}
	log.Debug().Msg("using memory idempotency store")
	ctx, cancel := context.WithCancel(context.Background())
	store := idempotency.NewMemoryStore(ctx)
	p.Lifecycle.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			cancel()
			return nil
		},
	})
	return store
}
