package di

import (
	"context"
	"fmt"

	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/pkg/utils"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/fx"
)

// MongoModule is the FX module for the MongoDB client
var MongoModule = fx.Module(
	"mongo",
	fx.Provide(
		provideMongoClient,
	),
	fx.Invoke(func(cfg *config.Config, mongoClient *mongo.Client) {
		if IsDriverEnabled(cfg.Database.Driver, mongoDriver) {
			if err := createCollectionsIndexes(cfg, mongoClient); err != nil {
				log.Fatal().Msgf("failed to create indexes for collections: %v", err)
			}
		}
	}),
)

// mongoCollections is the list of collections that will have indexes created
var mongoCollections = []string{
	repositories.GraphCollection,
	repositories.WorkflowCollection,
}

// provideMongoClient provides a MongoDB client if the driver is MongoDB, otherwise returns nil
func provideMongoClient(cfg *config.Config) *mongo.Client {
	log.Info().Msgf("providing mongo client, driver: %s", cfg.Database.Driver)

	if !IsDriverEnabled(cfg.Database.Driver, mongoDriver) {
		log.Info().Msgf("not providing mongo client, driver: %s", cfg.Database.Driver)
		return nil
	}

	uri := fmt.Sprintf("%s?authSource=%s", cfg.Database.URL, cfg.Database.AuthSource)
	clientOptions := options.Client().ApplyURI(uri)

	// To understand the options, see: https://www.mongodb.com/docs/drivers/go/current/fundamentals/bson/
	clientOptions.SetBSONOptions(&options.BSONOptions{
		UseJSONStructTags:   true,
		NilMapAsEmpty:       true,
		NilSliceAsEmpty:     true,
		NilByteSliceAsEmpty: true,
		OmitEmpty:           true,
	})

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

// createCollectionsIndexes creates indexes for the collections
func createCollectionsIndexes(
	cfg *config.Config,
	mongoClient *mongo.Client,
) error {
	dbName := utils.SerializeString(cfg.Database.Name)
	database := mongoClient.Database(dbName)

	for _, collectionName := range mongoCollections {
		log.Info().Msgf("creating indexes for collection: %s", collectionName)

		collection := database.Collection(collectionName)
		idxView := collection.Indexes()

		idxList, err := idxView.List(context.Background(), options.ListIndexes().SetBatchSize(100))
		if err != nil {
			log.Error().Msgf("failed to list indexes for collection: %s, error: %v", collectionName, err)
			return err
		}

		idxName := fmt.Sprintf("%s_id_idx", collectionName)

		for idxList.Next(context.Background()) {
			idx := bson.M{}
			if err := idxList.Decode(&idx); err != nil {
				log.Error().Msgf("failed to decode index for collection: %s, error: %v", collectionName, err)
				return err
			}
			log.Info().Msgf("index: %v", idx)
			if idx["name"] == idxName {
				log.Info().Msgf("index already exists for collection: %s", collectionName)
				continue
			}
		}

		opts := options.Index().SetUnique(true).SetName(idxName)

		// create the index
		_, err = idxView.CreateOne(context.Background(), mongo.IndexModel{
			Keys:    bson.M{"id": 1},
			Options: opts,
		})

		if err != nil {
			log.Error().Msgf("failed to create index for collection: %s, error: %v", collectionName, err)
			return err
		}

		log.Info().Msgf("index created for collection: %s", collectionName)
	}

	return nil
}
