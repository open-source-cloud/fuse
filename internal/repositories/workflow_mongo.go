package repositories

import (
	"errors"

	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/pkg/utils"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var (
	// ErrWorkflowMongoNotImplemented is returned when a workflow operation is not implemented
	ErrWorkflowMongoNotImplemented = errors.New("workflow mongo not implemented")
)

const (
	// WorkflowMongoCollection is the name of the collection in MongoDB
	WorkflowMongoCollection = "workflows"
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
	dbName := utils.SerializeString(config.Database.Name)
	database := client.Database(dbName)
	collection := database.Collection(WorkflowMongoCollection)

	return &MongoWorkflowRepository{
		config:     config,
		client:     client,
		database:   database,
		collection: collection,
	}
}

// Exists checks if a workflow exists in MongoDB
func (m *MongoWorkflowRepository) Exists(_ string) bool {
	return false
}

// Get retrieves a workflow from MongoDB
func (m *MongoWorkflowRepository) Get(_ string) (*workflow.Workflow, error) {
	return nil, ErrWorkflowMongoNotImplemented
}

// Save stores a workflow in MongoDB
func (m *MongoWorkflowRepository) Save(_ *workflow.Workflow) error {
	return ErrWorkflowMongoNotImplemented
}
