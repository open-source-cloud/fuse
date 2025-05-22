// Package repos Data repositories for the application
package repos

import (
	"fmt"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/pkg/store"
)

// NewMemoryGraphRepo creates a new in-memory GraphRepo
func NewMemoryGraphRepo() GraphRepo {
	return &memoryGraphRepo{
		graphs: store.New(),
	}
}

// GraphRepo workflow schema repository
type (
	GraphRepo interface {
		Get(id string) (*workflow.Graph, error)
		Save(graph *workflow.Graph) error
	}

	memoryGraphRepo struct {
		graphs *store.KV
	}
)

func (m *memoryGraphRepo) Get(id string) (*workflow.Graph, error) {
	foundGraph := m.graphs.Get(id)
	if foundGraph == nil {
		return nil, fmt.Errorf("graph %s not found", id)
	}

	return foundGraph.(*workflow.Graph), nil
}

func (m *memoryGraphRepo) Save(graph *workflow.Graph) error {
	m.graphs.Set(graph.ID(), graph)
	return nil
}
