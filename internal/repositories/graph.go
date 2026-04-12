// Package repositories data repositories for the application
package repositories

import (
	"errors"

	"github.com/open-source-cloud/fuse/internal/workflow"
)

var (
	// ErrGraphNotFound is returned when a graph is not found
	ErrGraphNotFound = errors.New("graph not found")
	// ErrGraphNotModified is returned when a graph is not modified
	ErrGraphNotModified = errors.New("graph not modified")
	// ErrSchemaVersionNotFound is returned when a specific schema version is not found
	ErrSchemaVersionNotFound = errors.New("schema version not found")
)

type (
	// GraphSchemaListItem is lightweight metadata for listing stored graph schemas.
	GraphSchemaListItem struct {
		SchemaID string
		Name     string
	}
	// GraphRepository defines the interface o a GraphRepository repository
	GraphRepository interface {
		FindByID(id string) (*workflow.Graph, error)
		Save(graph *workflow.Graph) error
		// List returns all stored schemas ordered by schema ID (ascending).
		List() ([]GraphSchemaListItem, error)

		// FindByIDAndVersion retrieves the graph for a specific schema version.
		FindByIDAndVersion(id string, version int) (*workflow.Graph, error)
		// SaveVersion persists a new SchemaVersion. If sv.IsActive is true the active-version
		// pointer for this schema is updated to sv.Version.
		SaveVersion(sv *workflow.SchemaVersion) error
		// ListVersions returns all recorded versions for a schema ordered by version ascending.
		ListVersions(schemaID string) ([]workflow.SchemaVersion, error)
		// SetActiveVersion updates the active version pointer and rebuilds the cached graph.
		SetActiveVersion(schemaID string, version int) error
		// GetVersionHistory returns aggregate version metadata for a schema.
		GetVersionHistory(schemaID string) (*workflow.SchemaVersionHistory, error)
	}
)
