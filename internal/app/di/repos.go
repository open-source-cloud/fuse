package di

import (
	"github.com/open-source-cloud/fuse/internal/app/config"
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
	),
)

// provideGraphRepository provides the appropriate GraphRepository based on config
func provideGraphRepository(cfg *config.Config, mongoClient *mongo.Client) repositories.GraphRepository {
	log.Debug().Msgf("using graph repository driver: %s", cfg.Database.Driver)

	if IsDriverEnabled(cfg.Database.Driver, mongoDriver) && mongoClient != nil {
		log.Debug().Msg("using mongodb graph repository")
		return repositories.NewMongoGraphRepository(mongoClient, cfg)
	}

	log.Debug().Msg("using memory graph repository")
	return repositories.NewMemoryGraphRepository()
}

// provideWorkflowRepository provides the appropriate WorkflowRepository based on config
func provideWorkflowRepository(cfg *config.Config, _ *mongo.Client) repositories.WorkflowRepository {
	log.Debug().Msgf("using workflow repository driver: %s", cfg.Database.Driver)

	// TODO: Temp disabled for testing
	// if IsDriverEnabled(cfg.Database.Driver, mongoDriver) && mongoClient != nil {
	// 	log.Debug().Msg("using mongodb workflow repository")
	// 	return repositories.NewMongoWorkflowRepository(mongoClient, cfg)
	// }

	log.Debug().Msg("using memory workflow repository")

	return repositories.NewMemoryWorkflowRepository()
}
