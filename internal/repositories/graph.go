// Package repositories data repositories for the application
package repositories

import (
	"errors"

	"github.com/open-source-cloud/fuse/internal/workflow"
)

var (
	ErrGraphNotFound = errors.New("graph not found")
)

type (
	// GraphRepository defines the interface o a GraphRepository repository
	GraphRepository interface {
		FindByID(id string) (*workflow.Graph, error)
		Save(graph *workflow.Graph) error
	}
)
