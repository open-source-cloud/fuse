package di

import (
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/fx"
)

// RepoModule FX module with the repo providers based on config
var RepoModule = fx.Module(
	"repo",
	fx.Provide(
		provideGraphRepository,
		provideWorkflowRepository,
		provideMongoClient,
	),
)

// provideGraphRepository provides the appropriate GraphRepository based on config
func provideGraphRepository(cfg *config.Config, mongoClient *mongo.Client) repositories.GraphRepository {
	log.Info().Msgf("using graph repository driver: %s", cfg.Database.Driver)

	switch cfg.Database.Driver {
	case mongodbDriver, mongoDriver:
		log.Info().Msg("using mongodb graph repository")
		return repositories.NewMongoGraphRepository(mongoClient, cfg)
	case memoryDriver, "":
		log.Info().Msg("using memory graph repository")
		return repositories.NewMemoryGraphRepository()
	default:
		return repositories.NewMemoryGraphRepository()
	}
}

// provideWorkflowRepository provides the appropriate WorkflowRepository based on config
func provideWorkflowRepository(cfg *config.Config, mongoClient *mongo.Client) repositories.WorkflowRepository {
	log.Info().Msgf("using workflow repository driver: %s", cfg.Database.Driver)

	switch cfg.Database.Driver {
	case mongodbDriver, mongoDriver:
		log.Info().Msg("using mongodb workflow repository")
		return repositories.NewMongoWorkflowRepository(mongoClient, cfg)
	case memoryDriver, "":
		log.Info().Msg("using memory workflow repository")
		return repositories.NewMemoryWorkflowRepository()
	default:
		return repositories.NewMemoryWorkflowRepository()
	}
}
