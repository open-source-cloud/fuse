package di

import (
	"context"
	"fmt"

	"github.com/open-source-cloud/fuse/internal/app/config"
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
	repositories.GraphMongoCollection,
	repositories.WorkflowMongoCollection,
	repositories.PackageMongoCollection,
}

// provideMongoClient provides a MongoDB client if the driver is MongoDB, otherwise returns nil
func provideMongoClient(cfg *config.Config) *mongo.Client {
	log.Debug().Msgf("providing mongo client, driver: %s", cfg.Database.Driver)

	if !IsDriverEnabled(cfg.Database.Driver, mongoDriver) {
		log.Debug().Msgf("not providing mongo client, driver: %s", cfg.Database.Driver)
		return nil
	}

	mongoCfg := cfg.Database.Mongo
	authSource := utils.SerializeString(mongoCfg.AuthSource)
	uri := cfg.Database.URL
	if authSource != "" {
		uri = fmt.Sprintf("%s?authSource=%s", uri, authSource)
	}
	clientOptions := options.Client().ApplyURI(uri)

	// To understand the options, see: https://www.mongodb.com/docs/drivers/go/current/fundamentals/bson/
	clientOptions.SetBSONOptions(&options.BSONOptions{
		UseJSONStructTags:   true,
		NilMapAsEmpty:       false,
		NilSliceAsEmpty:     false,
		NilByteSliceAsEmpty: false,
		OmitEmpty:           true,
	})

	log.Debug().Msg("connecting to mongodb")

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		log.Fatal().Msgf("Failed to connect to MongoDB: %v", err)
	}

	log.Debug().Msg("connected to mongodb")

	// Test the connection
	ctx := context.Background()
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal().Msgf("Failed to ping MongoDB: %v", err)
	}

	log.Debug().Msg("pinged mongodb")

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
		idxName := fmt.Sprintf("%s_id_idx", collectionName)
		log.Debug().Msgf("creating index for collection: %s, name: %s", collectionName, idxName)
		if err := createIndexIfNotExists(database.Collection(collectionName), idxName); err != nil {
			return err
		}
		log.Debug().Msgf("index created for collection: %s, idx: %s", collectionName, idxName)
	}

	return nil
}

// createIndexIfNotExists creates an index for a collection if it does not exist
func createIndexIfNotExists(
	collection *mongo.Collection,
	idxName string,
) error {
	idxView := collection.Indexes()

	opts := options.Index().SetUnique(true).SetName(idxName)

	_, err := idxView.CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.M{"id": 1},
		Options: opts,
	})

	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			log.Debug().Msgf("index already exists for collection: %s", collection.Name())
			return nil
		}
		return fmt.Errorf("failed to create index for collection: %s, error: %v", collection.Name(), err)
	}

	return nil
}
