package di

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/open-source-cloud/fuse/internal/repositories/postgres"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
)

// DatabaseModule provides the PostgreSQL connection pool and optional PG listener.
var DatabaseModule = fx.Module(
	"database",
	fx.Provide(providePgxPool),
	fx.Provide(providePgListener),
)

// pgxPoolResult wraps the pool so fx can provide a nil *pgxpool.Pool when driver=memory.
type pgxPoolResult struct {
	fx.Out
	Pool *pgxpool.Pool `optional:"true"`
}

func providePgxPool(lc fx.Lifecycle, cfg *config.Config) (pgxPoolResult, error) {
	if cfg.Database.Driver != config.DBDriverPostgres {
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
			if err := pool.Ping(ctx); err != nil {
				return err
			}
			log.Info().Msg("running database migrations...")
			return postgres.RunMigrations(cfg.Database.PostgresDSN)
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

// pgListenerResult wraps the PgListener so fx can provide nil when not needed.
type pgListenerResult struct {
	fx.Out
	Listener *postgres.PgListener `optional:"true"`
}

func providePgListener(cfg *config.Config) (pgListenerResult, error) {
	if cfg.Database.Driver != config.DBDriverPostgres || !cfg.HA.Enabled {
		log.Debug().Msg("PG listener not needed (driver != postgres or HA disabled)")
		return pgListenerResult{}, nil
	}

	if cfg.Database.PostgresDSN == "" {
		return pgListenerResult{}, nil
	}

	listener, err := postgres.NewPgListener(context.Background(), cfg.Database.PostgresDSN)
	if err != nil {
		return pgListenerResult{}, err
	}

	log.Info().Msg("PG LISTEN/NOTIFY listener created")
	return pgListenerResult{Listener: listener}, nil
}
