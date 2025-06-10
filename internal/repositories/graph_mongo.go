package repositories

import (
	"context"
	"fmt"

	"github.com/open-source-cloud/fuse/internal/workflow"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type MongoGraphRepository struct {
	client     *mongo.Client
	database   *mongo.Database
	collection *mongo.Collection
}

func NewMongoGraphRepository(client *mongo.Client) GraphRepository {
	database := client.Database("fuse")
	collection := database.Collection("graphs")

	return &MongoGraphRepository{
		client:     client,
		database:   database,
		collection: collection,
	}
}

// FindByID retrieves a graph from MongoDB
func (m *MongoGraphRepository) FindByID(id string) (*workflow.Graph, error) {
	ctx := context.Background()

	var result bson.M
	err := m.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("graph %s not found", id)
		}
		return nil, fmt.Errorf("failed to find graph %s: %w", id, err)
	}

	// TODO: Implement proper BSON to Graph conversion
	// For now, this is a placeholder - you'll need to implement proper serialization
	return nil, fmt.Errorf("MongoDB FindByID not fully implemented yet")
}

// Save stores a graph in MongoDB
func (m *MongoGraphRepository) Save(graph *workflow.Graph) error {
	ctx := context.Background()

	// TODO: Implement proper Graph to BSON conversion
	// For now, this is a placeholder - you'll need to implement proper serialization
	document := bson.M{
		"_id": graph.ID(),
		// Add other graph fields here
	}

	_, err := m.collection.ReplaceOne(
		ctx,
		bson.M{"_id": graph.ID()},
		document,
		nil, // Use default ReplaceOptions
	)
	if err != nil {
		return fmt.Errorf("failed to save graph %s: %w", graph.ID(), err)
	}

	return nil
}
