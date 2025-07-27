package repositories

import (
	"context"

	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/pkg/utils"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	// GraphMongoCollection is the name of the collection in MongoDB
	GraphMongoCollection = "graphs"
)

// MongoGraphRepository is a MongoDB implementation of the GraphRepository interface
type MongoGraphRepository struct {
	config     *config.Config
	client     *mongo.Client
	database   *mongo.Database
	collection *mongo.Collection
}

// NewMongoGraphRepository creates a new MongoDB GraphRepository
func NewMongoGraphRepository(client *mongo.Client, config *config.Config) GraphRepository {
	dbName := utils.SerializeString(config.Database.Name)
	database := client.Database(dbName)
	collection := database.Collection(GraphMongoCollection)
	return &MongoGraphRepository{
		config:     config,
		client:     client,
		database:   database,
		collection: collection,
	}
}

// FindByID retrieves a graph from MongoDB
func (m *MongoGraphRepository) FindByID(id string) (*workflow.Graph, error) {
	ctx := context.Background()

	var schema *workflow.GraphSchema
	err := m.collection.FindOne(ctx, bson.M{"id": id}).Decode(&schema)
	if err != nil {
		log.Error().Msgf("failed to find graph %s: %v", id, err)
		if err == mongo.ErrNoDocuments {
			return nil, ErrGraphNotFound
		}
		return nil, err
	}

	graph, err := workflow.NewGraph(schema)
	if err != nil {
		log.Error().Msgf("failed to create graph from schema: %v", err)
		return nil, err
	}

	return graph, nil
}

// Save stores a graph in MongoDB
func (m *MongoGraphRepository) Save(graph *workflow.Graph) error {
	ctx := context.Background()

	log.Info().Msgf("saving graph %s", graph.ID())

	// get a deep copy of the schema of the graph
	schema := graph.Schema()

	result, err := m.collection.ReplaceOne(
		ctx,
		bson.M{"id": schema.ID},
		schema,
		options.Replace().SetUpsert(true),
	)
	if err != nil {
		log.Error().Msgf("failed to save graph %s: %v", schema.ID, err)
		return err
	}

	if result.UpsertedCount > 0 {
		log.Info().Msgf("graph %s upserted", schema.ID)
		return nil
	}

	if result.ModifiedCount > 0 {
		log.Info().Msgf("graph %s modified", schema.ID)
		return nil
	}

	return ErrGraphNotModified
}
