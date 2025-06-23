package services

import (
	"errors"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/internal/workflow"
)

// GraphSchemaService represents the transactional and logical service to manage workflow.Graph
type GraphSchemaService struct {
	graphRepo    repositories.GraphRepository
	graphFactory *workflow.GraphFactory
}

// NewGraphSchemaService returns a new GraphSchemaService
func NewGraphSchemaService(graphRepo repositories.GraphRepository, graphFactory *workflow.GraphFactory) *GraphSchemaService {
	return &GraphSchemaService{
		graphRepo:    graphRepo,
		graphFactory: graphFactory,
	}
}

// Upsert upserts a workflow.GraphSchema
func (gs *GraphSchemaService) Upsert(schemaID string, incomingSchema *workflow.GraphSchema) error {
	graph, err := gs.graphRepo.FindByID(schemaID)

	if err != nil {
		if errors.As(err, &repositories.ErrGraphNotFound) {
			return gs.Create(schemaID, incomingSchema)
		}
		return err
	}

	return gs.Update(graph.Schema())
}

// Create creates and save a new instance of workflow.GraphSchema at database
func (gs *GraphSchemaService) Create(schemaID string, schema *workflow.GraphSchema) error {
	if schema.ID == "" {
		schema.ID = schemaID
	}

	if err := schema.Validate(); err != nil {
		return err
	}

	graph, err := gs.graphFactory.NewGraphFromSchema(schema)
	if err != nil {
		return err
	}

	if err := gs.graphRepo.Save(graph); err != nil {
		return err
	}

	return nil
}

// Update updates a workflow.GraphSchema into database
func (gs *GraphSchemaService) Update(schema *workflow.GraphSchema) error {
	return nil
}
