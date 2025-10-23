package repositories

import (
	"context"

	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/open-source-cloud/fuse/pkg/strutil"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	// PackageMongoCollection is the name of the collection in MongoDB
	PackageMongoCollection = "packages"
)

type (
	// MongoPackageRepository is a MongoDB implementation of the PackageRepository interface
	MongoPackageRepository struct {
		PackageRepository
		client     *mongo.Client
		database   *mongo.Database
		collection *mongo.Collection
	}
)

// NewMongoPackageRepository creates a new MongoDB PackageRepository
func NewMongoPackageRepository(client *mongo.Client, config *config.Config) PackageRepository {
	dbName := strutil.SerializeString(config.Database.Name)
	database := client.Database(dbName)
	collection := database.Collection(PackageMongoCollection)
	return &MongoPackageRepository{
		client:     client,
		database:   database,
		collection: collection,
	}
}

// FindByID finds a package by ID in MongoDB
func (m *MongoPackageRepository) FindByID(id string) (*workflow.Package, error) {
	filter := bson.M{"id": id}
	var pkg workflow.Package
	err := m.collection.FindOne(context.Background(), filter).Decode(&pkg)
	if err != nil {
		log.Error().Str("packageID", id).Msgf("failed to find package: %v", err)
		if err == mongo.ErrNoDocuments {
			return nil, ErrPackageNotFound
		}
		return nil, err
	}
	return &pkg, nil
}

// FindAll finds all packages in MongoDB
func (m *MongoPackageRepository) FindAll() ([]*workflow.Package, error) {
	cursor, err := m.collection.Find(context.Background(), bson.M{})
	if err != nil {
		log.Error().Msgf("failed to find all packages: %v", err)
		return nil, err
	}

	var packages []*workflow.Package
	if err := cursor.All(context.Background(), &packages); err != nil {
		log.Error().Msgf("failed to decode packages: %v", err)
		return nil, err
	}

	// if no packages are found, return an empty slice
	if len(packages) == 0 {
		return []*workflow.Package{}, nil
	}

	return packages, nil
}

// Save saves a package to MongoDB
func (m *MongoPackageRepository) Save(pkg *workflow.Package) error {
	ctx := context.Background()

	log.Info().Msgf("saving package %s", pkg.ID)

	result, err := m.collection.ReplaceOne(
		ctx,
		bson.M{"id": pkg.ID},
		pkg,
		options.Replace().SetUpsert(true),
	)
	if err != nil {
		log.Error().Msgf("failed to save package %s: %v", pkg.ID, err)
		return err
	}

	if result.UpsertedCount > 0 {
		log.Info().Msgf("package %s upserted", pkg.ID)
		return nil
	}

	if result.ModifiedCount > 0 {
		log.Info().Msgf("package %s modified", pkg.ID)
		return nil
	}

	return ErrPackageNotModified
}

// Delete deletes a package from MongoDB
func (m *MongoPackageRepository) Delete(id string) error {
	ctx := context.Background()

	log.Info().Msgf("deleting package %s", id)

	_, err := m.collection.DeleteOne(ctx, bson.M{"id": id})
	if err != nil {
		log.Error().Msgf("failed to delete package %s: %v", id, err)
		return err
	}

	return nil
}
