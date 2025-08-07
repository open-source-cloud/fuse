// Package services provide the services for the application
package services

import (
	"errors"
	"strings"

	"github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/rs/zerolog/log"
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
	graph, err := gs.graphRepo.FindByID(schemaID)
	if err != nil {
		return nil, err
	}

	// populate the metadata of the graph's nodes if the nodes are not set
	if !graph.IsNodesMetadataPopulated() {
		log.Warn().Msgf("graph's %s nodes metadata is not populated, populating...", graph.ID())
		if err := gs.populateNodeMetadata(graph, nil); err != nil {
			return nil, err
		}
	}

	return graph, nil
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
	log.Info().Msgf("populating node metadata for graph %s with %d nodes", graph.ID(), len(nodes))

	// if the package registry is not set, return nil
	// in the mermaid flowchart; the nodes are not populated with the metadata
	if gs.packageRegistry == nil {
		log.Warn().Msg("package registry is not set, skipping node metadata population")
		return nil
	}

	// if the nodes are not set, use the schema's nodes to populate the metadata
	if len(nodes) == 0 {
		schema := graph.Schema()
		nodes = schema.Nodes
	}

	for _, node := range nodes {
		log.Info().Msgf("populating node metadata for node %s", node.ID)

		lastIndexOfSlash := strings.LastIndex(node.Function, "/")
		if lastIndexOfSlash == -1 {
			log.Error().Msgf("invalid function format '%s': must contain '/' to separate package and function", node.Function)
			return workflow.ErrInvalidFunctionFormat
		}
		pkgID := node.Function[:lastIndexOfSlash]
		if pkgID == "" {
			log.Error().Msgf("invalid function format '%s': must contain '/' to separate package and function", node.Function)
			return workflow.ErrInvalidFunctionFormat
		}
		pkg, err := gs.packageRegistry.Get(pkgID)
		if err != nil {
			log.Error().Err(err).Msgf("failed to get package %s metadata for node %s", pkgID, node.ID)
			return err
		}
		pkgFnMetadata, err := pkg.GetFunctionMetadata(node.Function)
		if err != nil {
			log.Error().Err(err).Msgf("failed to get function %s metadata for node %s", node.Function, node.ID)
			return err
		}
		log.Debug().Msgf("updating node %s metadata", node.ID)
		if err := graph.UpdateNodeMetadata(node.ID, pkgFnMetadata); err != nil {
			log.Error().Err(err).Msgf("failed to update node %s metadata", node.ID)
			return err
		}
		log.Debug().Msgf("updated node %s metadata", node.ID)
	}

	log.Info().Msgf("populated node metadata for graph %s with %d nodes", graph.ID(), len(nodes))

	return nil
}
