package di

import (
	"context"
	"slices"
	"strings"

	"github.com/open-source-cloud/fuse/app/config"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// serializeString serializes a string by converting it to lowercase and trimming whitespace
func serializeString(str string) string {
	if str == "" {
		return ""
	}
	str = strings.ToLower(str)
	str = strings.TrimSpace(str)
	return str
}

// provideMongoClient provides a MongoDB client if the driver is MongoDB, otherwise returns nil
func provideMongoClient(cfg *config.Config) *mongo.Client {
	log.Info().Msgf("providing mongo client, driver: %s", cfg.Database.Driver)

	allowedDrivers := []string{mongodbDriver, mongoDriver}

	driver := serializeString(cfg.Database.Driver)

	if !slices.Contains(allowedDrivers, driver) {
		log.Info().Msgf("not providing mongo client, driver: %s", driver)
		return nil
	}

	log.Debug().Msgf("connection string: %s", cfg.Database.URL)

	clientOptions := options.Client().ApplyURI(cfg.Database.URL)

	log.Info().Msg("connecting to mongodb")

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		log.Fatal().Msgf("Failed to connect to MongoDB: %v", err)
	}

	log.Info().Msg("connected to mongodb")

	// Test the connection
	ctx := context.Background()
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal().Msgf("Failed to ping MongoDB: %v", err)
	}

	log.Info().Msg("pinged mongodb")

	return client
}
