package repositories

import (
	"context"
	"fmt"

	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const (
	// workflowCollection is the name of the collection in MongoDB
	workflowCollection = "workflows"
)

// MongoWorkflowRepository is a MongoDB implementation of the WorkflowRepository interface
type MongoWorkflowRepository struct {
	WorkflowRepository

	config     *config.Config
	client     *mongo.Client
	database   *mongo.Database
	collection *mongo.Collection
}

// NewMongoWorkflowRepository creates a new MongoDB WorkflowRepository
func NewMongoWorkflowRepository(client *mongo.Client, config *config.Config) WorkflowRepository {
	database := client.Database(config.Database.Name)
	collection := database.Collection(workflowCollection)

	return &MongoWorkflowRepository{
		config:     config,
		client:     client,
		database:   database,
		collection: collection,
	}
}

// Exists checks if a workflow exists in MongoDB
func (m *MongoWorkflowRepository) Exists(id string) bool {
	ctx := context.Background()

	count, err := m.collection.CountDocuments(ctx, bson.M{"_id": id})
	if err != nil {
		return false
	}

	return count > 0
}

// Get retrieves a workflow from MongoDB
func (m *MongoWorkflowRepository) Get(id string) (*workflow.Workflow, error) {
	ctx := context.Background()

	var result bson.M
	err := m.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("workflow %s not found", id)
		}
		return nil, fmt.Errorf("failed to find workflow %s: %w", id, err)
	}

	// TODO: Implement proper BSON to Workflow conversion
	// For now, this is a placeholder - you'll need to implement proper serialization
	return nil, fmt.Errorf("MongoDB Get not fully implemented yet")
}

// Save stores a workflow in MongoDB
func (m *MongoWorkflowRepository) Save(workflow *workflow.Workflow) error {
	ctx := context.Background()

	// TODO: Implement proper Workflow to BSON conversion
	// For now, this is a placeholder - you'll need to implement proper serialization
	document := bson.M{
		"_id": workflow.ID().String(),
		// Add other workflow fields here
	}

	_, err := m.collection.ReplaceOne(
		ctx,
		bson.M{"_id": workflow.ID().String()},
		document,
		nil, // Use default ReplaceOptions
	)
	if err != nil {
		return fmt.Errorf("failed to save workflow %s: %w", workflow.ID().String(), err)
	}

	return nil
}
