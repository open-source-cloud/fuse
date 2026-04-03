package di

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
)

// DatabaseModule provides the PostgreSQL connection pool when DB_DRIVER=postgres.
var DatabaseModule = fx.Module(
	"database",
	fx.Provide(providePgxPool),
)

// pgxPoolResult wraps the pool so fx can provide a nil *pgxpool.Pool when driver=memory.
type pgxPoolResult struct {
	fx.Out
	Pool *pgxpool.Pool `optional:"true"`
}

func providePgxPool(lc fx.Lifecycle, cfg *config.Config) (pgxPoolResult, error) {
	if cfg.Database.Driver != "postgres" {
		log.Debug().Msg("database driver is not postgres, skipping pool creation")
		return pgxPoolResult{}, nil
	}

	if cfg.Database.PostgresDSN == "" {
		log.Warn().Msg("DB_DRIVER=postgres but DB_POSTGRES_DSN is empty")
		return pgxPoolResult{}, nil
	}

	poolCfg, err := pgxpool.ParseConfig(cfg.Database.PostgresDSN)
	if err != nil {
		return pgxPoolResult{}, err
	}
	poolCfg.MaxConns = cfg.Database.MaxOpenConns
	poolCfg.MinConns = cfg.Database.MaxIdleConns
	poolCfg.MaxConnLifetime = cfg.Database.ConnMaxLifetime

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return pgxPoolResult{}, err
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info().Msg("pinging PostgreSQL...")
			return pool.Ping(ctx)
		},
		OnStop: func(_ context.Context) error {
			log.Info().Msg("closing PostgreSQL connection pool")
			pool.Close()
			return nil
		},
	})

	log.Info().Str("dsn", cfg.Database.PostgresDSN).Msg("PostgreSQL pool created")
	return pgxPoolResult{Pool: pool}, nil
}
