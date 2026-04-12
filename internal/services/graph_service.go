// Package services provide the services for the application
package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/rs/zerolog/log"
)

type (
	// GraphService represents the transactional and logical service to manage workflow.Graph
	GraphService interface {
		FindByID(schemaID string) (*workflow.Graph, error)
		// FindByIDAndVersion retrieves the graph for a specific schema version.
		FindByIDAndVersion(schemaID string, version int) (*workflow.Graph, error)
		// ListSchemas returns lightweight metadata for every stored graph schema.
		ListSchemas() ([]repositories.GraphSchemaListItem, error)
		// ListVersions returns all recorded versions for a schema.
		ListVersions(schemaID string) ([]workflow.SchemaVersion, error)
		// GetVersionHistory returns aggregate version metadata for a schema.
		GetVersionHistory(schemaID string) (*workflow.SchemaVersionHistory, error)
		// Upsert creates or updates a schema, always creating a new version.
		Upsert(schemaID string, schema *workflow.GraphSchema) (*workflow.Graph, error)
		// SetActiveVersion activates a specific existing version of a schema.
		SetActiveVersion(schemaID string, version int) error
		// Rollback creates a new version with the content of an older version and activates it.
		Rollback(schemaID string, toVersion int, comment string) (*workflow.SchemaVersion, error)
		// ApplyReplicatedUpsert applies a schema from a peer cluster event (does not republish).
		ApplyReplicatedUpsert(schemaID string, schemaJSON []byte) error
		// EnsureNodeMetadata populates function metadata on graph nodes if not already present.
		// This is needed when a graph is loaded from persistence without package registry access.
		EnsureNodeMetadata(graph *workflow.Graph) error
	}
	// DefaultGraphService is the default implementation of the GraphService interface
	DefaultGraphService struct {
		graphRepo       repositories.GraphRepository
		packageRegistry packages.Registry
		publisher       SchemaUpsertPublisher
	}
)

// NewGraphService returns a new GraphService
func NewGraphService(
	graphRepo repositories.GraphRepository,
	packageRegistry packages.Registry,
	publisher SchemaUpsertPublisher,
) GraphService {
	return &DefaultGraphService{
		graphRepo:       graphRepo,
		packageRegistry: packageRegistry,
		publisher:       publisher,
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

// FindByIDAndVersion retrieves the graph for a specific schema version.
func (gs *DefaultGraphService) FindByIDAndVersion(schemaID string, version int) (*workflow.Graph, error) {
	graph, err := gs.graphRepo.FindByIDAndVersion(schemaID, version)
	if err != nil {
		return nil, err
	}

	if !graph.IsNodesMetadataPopulated() {
		if err := gs.populateNodeMetadata(graph, nil); err != nil {
			return nil, err
		}
	}

	return graph, nil
}

// ListSchemas lists all stored graph schemas from the repository.
func (gs *DefaultGraphService) ListSchemas() ([]repositories.GraphSchemaListItem, error) {
	return gs.graphRepo.List()
}

// ListVersions returns all recorded versions for a schema.
func (gs *DefaultGraphService) ListVersions(schemaID string) ([]workflow.SchemaVersion, error) {
	return gs.graphRepo.ListVersions(schemaID)
}

// GetVersionHistory returns aggregate version metadata for a schema.
func (gs *DefaultGraphService) GetVersionHistory(schemaID string) (*workflow.SchemaVersionHistory, error) {
	return gs.graphRepo.GetVersionHistory(schemaID)
}

// Upsert upserts a workflow.GraphSchema into the database, creating a new version on each call.
func (gs *DefaultGraphService) Upsert(schemaID string, schema *workflow.GraphSchema) (*workflow.Graph, error) {
	g, err := gs.upsertGraph(schemaID, schema)
	if err != nil {
		return nil, err
	}
	if gs.publisher != nil {
		pubSchema := g.Schema()
		gs.publisher.PublishLocalUpsert(pubSchema.ID, &pubSchema)
	}
	return g, nil
}

// SetActiveVersion activates a specific existing version of a schema.
func (gs *DefaultGraphService) SetActiveVersion(schemaID string, version int) error {
	_, err := gs.graphRepo.FindByIDAndVersion(schemaID, version)
	if err != nil {
		if errors.Is(err, repositories.ErrSchemaVersionNotFound) {
			return repositories.ErrSchemaVersionNotFound
		}
		return err
	}
	return gs.graphRepo.SetActiveVersion(schemaID, version)
}

// Rollback creates a new version with the content of an older version and activates it.
func (gs *DefaultGraphService) Rollback(schemaID string, toVersion int, comment string) (*workflow.SchemaVersion, error) {
	oldGraph, err := gs.graphRepo.FindByIDAndVersion(schemaID, toVersion)
	if err != nil {
		if errors.Is(err, repositories.ErrSchemaVersionNotFound) {
			return nil, fmt.Errorf("version %d not found for schema %s: %w", toVersion, schemaID, repositories.ErrSchemaVersionNotFound)
		}
		return nil, err
	}

	history, err := gs.graphRepo.GetVersionHistory(schemaID)
	if err != nil {
		return nil, err
	}

	newVersionNum := history.LatestVersion + 1
	oldSchema := oldGraph.Schema()

	// Build and populate the graph so FindByID returns a metadata-populated graph
	newGraph, err := workflow.NewGraph(&oldSchema)
	if err != nil {
		return nil, fmt.Errorf("rollback: rebuild graph: %w", err)
	}
	if err := gs.populateNodeMetadata(newGraph, oldSchema.Nodes); err != nil {
		return nil, fmt.Errorf("rollback: populate node metadata: %w", err)
	}
	if err := gs.graphRepo.Save(newGraph); err != nil {
		return nil, fmt.Errorf("rollback: save graph: %w", err)
	}

	sv := &workflow.SchemaVersion{
		SchemaID:  schemaID,
		Version:   newVersionNum,
		Schema:    oldSchema,
		CreatedAt: time.Now().UTC(),
		Comment:   comment,
		IsActive:  true,
	}
	if err := gs.graphRepo.SaveVersion(sv); err != nil {
		return nil, fmt.Errorf("rollback: save version: %w", err)
	}

	return sv, nil
}

// ApplyReplicatedUpsert applies JSON from a peer without emitting another replication event.
func (gs *DefaultGraphService) ApplyReplicatedUpsert(schemaID string, schemaJSON []byte) error {
	schema, err := workflow.NewGraphSchemaFromJSON(schemaJSON)
	if err != nil {
		return err
	}
	_, err = gs.upsertGraph(schemaID, schema)
	return err
}

func (gs *DefaultGraphService) upsertGraph(schemaID string, schema *workflow.GraphSchema) (*workflow.Graph, error) {
	// API / replication event key is canonical: must match trigger and GET /schemas/{id}.
	if schemaID != "" {
		schema.ID = schemaID
	}
	if schema.ID == "" {
		return nil, errors.New("graph schema id is required")
	}

	graph, err := gs.graphRepo.FindByID(schema.ID)
	if err != nil {
		if errors.Is(err, repositories.ErrGraphNotFound) {
			return gs.createVersioned(schema)
		}
		return nil, err
	}

	if graph == nil {
		return gs.createVersioned(schema)
	}

	return gs.updateVersioned(graph, schema)
}

// createVersioned creates a new graph and records it as version 1.
func (gs *DefaultGraphService) createVersioned(schema *workflow.GraphSchema) (*workflow.Graph, error) {
	graph, err := workflow.NewGraph(schema)
	if err != nil {
		return nil, err
	}

	if err := gs.populateNodeMetadata(graph, schema.Nodes); err != nil {
		return nil, err
	}

	if err := gs.graphRepo.Save(graph); err != nil {
		return nil, err
	}

	sv := &workflow.SchemaVersion{
		SchemaID:  schema.ID,
		Version:   1,
		Schema:    schema.Clone(),
		CreatedAt: time.Now().UTC(),
		IsActive:  true,
	}
	if err := gs.graphRepo.SaveVersion(sv); err != nil {
		return nil, err
	}

	return graph, nil
}

// updateVersioned updates an existing graph and records it as the next version.
func (gs *DefaultGraphService) updateVersioned(graph *workflow.Graph, schema *workflow.GraphSchema) (*workflow.Graph, error) {
	if err := schema.Validate(); err != nil {
		return nil, err
	}

	// Determine next version number before mutating the graph
	history, err := gs.graphRepo.GetVersionHistory(schema.ID)
	if err != nil && !errors.Is(err, repositories.ErrGraphNotFound) {
		return nil, err
	}

	newVersionNum := 1
	if history != nil && history.LatestVersion > 0 {
		newVersionNum = history.LatestVersion + 1
	}

	if err := graph.UpdateSchema(schema); err != nil {
		return nil, err
	}

	if err := gs.populateNodeMetadata(graph, schema.Nodes); err != nil {
		return nil, err
	}

	if err := gs.graphRepo.Save(graph); err != nil {
		return nil, err
	}

	sv := &workflow.SchemaVersion{
		SchemaID:  schema.ID,
		Version:   newVersionNum,
		Schema:    schema.Clone(),
		CreatedAt: time.Now().UTC(),
		IsActive:  true,
	}
	if err := gs.graphRepo.SaveVersion(sv); err != nil {
		return nil, err
	}

	return graph, nil
}

// EnsureNodeMetadata populates function metadata on graph nodes if not already present.
func (gs *DefaultGraphService) EnsureNodeMetadata(graph *workflow.Graph) error {
	if graph.IsNodesMetadataPopulated() {
		return nil
	}
	return gs.populateNodeMetadata(graph, nil)
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
