package di

import (
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
)

// RepoModule wires in-memory repositories only. Workflow, graph, package, and journal
// state are held in process memory (no external persistence layer).
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

func provideGraphRepository() repositories.GraphRepository {
	log.Debug().Msg("using memory graph repository")
	return repositories.NewMemoryGraphRepository()
}

func provideWorkflowRepository() repositories.WorkflowRepository {
	log.Debug().Msg("using memory workflow repository")
	return repositories.NewMemoryWorkflowRepository()
}

func provideJournalRepository() repositories.JournalRepository {
	log.Debug().Msg("using memory journal repository")
	return repositories.NewMemoryJournalRepository()
}

func providePackageRepository() repositories.PackageRepository {
	log.Debug().Msg("using memory package repository")
	return repositories.NewMemoryPackageRepository()
}

func provideAwakeableRepository() repositories.AwakeableRepository {
	log.Debug().Msg("using memory awakeable repository")
	return repositories.NewMemoryAwakeableRepository()
}
