package di

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/internal/repositories/postgres"
	"github.com/open-source-cloud/fuse/pkg/objectstore"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
)

// RepoModule provides all five repository interfaces, selecting implementations
// based on the DB_DRIVER config (memory or postgres).
var RepoModule = fx.Module(
	"repo",
	fx.Provide(
		provideGraphRepository,
		provideWorkflowRepository,
		providePackageRepository,
		provideJournalRepository,
		provideAwakeableRepository,
	),
)

type repoParams struct {
	fx.In
	Config *config.Config
	Pool   *pgxpool.Pool       `optional:"true"`
	Store  objectstore.ObjectStore
}

func provideGraphRepository(p repoParams) repositories.GraphRepository {
	if p.Config.Database.Driver == config.DBDriverPostgres && p.Pool != nil {
		log.Debug().Msg("using postgres graph repository")
		return postgres.NewGraphRepository(p.Pool, p.Store)
	}
	log.Debug().Msg("using memory graph repository")
	return repositories.NewMemoryGraphRepository()
}

func provideWorkflowRepository(p repoParams) repositories.WorkflowRepository {
	if p.Config.Database.Driver == config.DBDriverPostgres && p.Pool != nil {
		log.Debug().Msg("using postgres workflow repository")
		return postgres.NewWorkflowRepository(p.Pool, p.Store)
	}
	log.Debug().Msg("using memory workflow repository")
	return repositories.NewMemoryWorkflowRepository()
}

func provideJournalRepository(p repoParams) repositories.JournalRepository {
	if p.Config.Database.Driver == config.DBDriverPostgres && p.Pool != nil {
		log.Debug().Msg("using postgres journal repository")
		return postgres.NewJournalRepository(p.Pool, p.Store)
	}
	log.Debug().Msg("using memory journal repository")
	return repositories.NewMemoryJournalRepository()
}

func providePackageRepository(p repoParams) repositories.PackageRepository {
	if p.Config.Database.Driver == config.DBDriverPostgres && p.Pool != nil {
		log.Debug().Msg("using postgres package repository")
		return postgres.NewPackageRepository(p.Pool, p.Store)
	}
	log.Debug().Msg("using memory package repository")
	return repositories.NewMemoryPackageRepository()
}

func provideAwakeableRepository(p repoParams) repositories.AwakeableRepository {
	if p.Config.Database.Driver == config.DBDriverPostgres && p.Pool != nil {
		log.Debug().Msg("using postgres awakeable repository")
		return postgres.NewAwakeableRepository(p.Pool, p.Store)
	}
	log.Debug().Msg("using memory awakeable repository")
	return repositories.NewMemoryAwakeableRepository()
}
