// Package services provide the services for the application
package services

import (
	"errors"
	"strings"

	"github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/internal/workflow"
)

type (
	// GraphService represents the transactional and logical service to manage workflow.Graph
	GraphService interface {
		FindByID(schemaID string) (*workflow.Graph, error)
		Upsert(schemaID string, schema *workflow.GraphSchema) (*workflow.Graph, error)
	}
	// DefaultGraphService is the default implementation of the GraphService interface
	DefaultGraphService struct {
		graphRepo       repositories.GraphRepository
		packageRegistry packages.Registry
	}
)

// NewGraphService returns a new GraphService
func NewGraphService(graphRepo repositories.GraphRepository, packageRegistry packages.Registry) GraphService {
	return &DefaultGraphService{
		graphRepo:       graphRepo,
		packageRegistry: packageRegistry,
	}
}

// FindByID finds a workflow.GraphSchema by ID
func (gs *DefaultGraphService) FindByID(schemaID string) (*workflow.Graph, error) {
	return gs.graphRepo.FindByID(schemaID)
}

// Upsert upserts a workflow.GraphSchema into the database
func (gs *DefaultGraphService) Upsert(schemaID string, schema *workflow.GraphSchema) (*workflow.Graph, error) {
	if schema.ID == "" {
		schema.ID = schemaID
	}

	// check if the graph already exists
	graph, err := gs.graphRepo.FindByID(schema.ID)
	if err != nil {
		if errors.Is(err, repositories.ErrGraphNotFound) {
			return gs.create(schema)
		}
		return nil, err
	}

	// redundant check, but just in case
	if graph == nil {
		return gs.create(schema)
	}

	return gs.update(graph, schema)
}

// Update updates a workflow.GraphSchema into the database
func (gs *DefaultGraphService) update(graph *workflow.Graph, schema *workflow.GraphSchema) (*workflow.Graph, error) {
	if err := schema.Validate(); err != nil {
		return nil, err
	}

	if err := graph.UpdateSchema(schema); err != nil {
		return nil, err
	}

	// populate the metadata of the graph's nodes
	if err := gs.populateNodeMetadata(graph, schema.Nodes); err != nil {
		return nil, err
	}

	if err := gs.graphRepo.Save(graph); err != nil {
		return nil, err
	}

	return graph, nil
}

// create creates a workflow.GraphSchema in the database
func (gs *DefaultGraphService) create(schema *workflow.GraphSchema) (*workflow.Graph, error) {
	graph, err := workflow.NewGraph(schema)
	if err != nil {
		return nil, err
	}

	// populate the metadata of the graph's nodes
	if err := gs.populateNodeMetadata(graph, schema.Nodes); err != nil {
		return nil, err
	}

	if err := gs.graphRepo.Save(graph); err != nil {
		return nil, err
	}

	return graph, nil
}

// populateNodeMetadata populates the metadata of the graph's nodes
func (gs *DefaultGraphService) populateNodeMetadata(graph *workflow.Graph, nodes []*workflow.NodeSchema) error {
	// if the package registry is not set, return nil
	// in the mermaid flowchart; the nodes are not populated with the metadata
	if gs.packageRegistry == nil {
		return nil
	}

	for _, node := range nodes {
		lastIndexOfSlash := strings.LastIndex(node.Function, "/")
		pkgID := node.Function[:lastIndexOfSlash]
		pkg, err := gs.packageRegistry.Get(pkgID)
		if err != nil {
			return err
		}
		pkgFnMetadata, err := pkg.GetFunctionMetadata(node.Function)
		if err != nil {
			return err
		}
		if err := graph.UpdateNodeMetadata(node.ID, pkgFnMetadata); err != nil {
			return err
		}
	}

	return nil
}
