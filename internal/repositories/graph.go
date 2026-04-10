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
	}
)
