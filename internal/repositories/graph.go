// Package repositories data repositories for the application
package repositories

import (
	"fmt"
	"sync"

	"github.com/open-source-cloud/fuse/internal/workflow"
)

// NewMemoryGraphRepository creates a new in-memory GraphRepository
func NewMemoryGraphRepository() GraphRepository {
	return &DefaultGraphRepository{
		graphs: make(map[string]*workflow.Graph),
	}
}

type (
	// GraphRepository defines the interface o a GraphRepository repository
	GraphRepository interface {
		FindByID(id string) (*workflow.Graph, error)
		Save(graph *workflow.Graph) error
	}
	// DefaultGraphRepository is the default implementation of the GraphRepository interface (in-memory)
	DefaultGraphRepository struct {
		mu     sync.RWMutex
		graphs map[string]*workflow.Graph
	}
)

// FindByID retrieves a graph from the repository
func (m *DefaultGraphRepository) FindByID(id string) (*workflow.Graph, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	graph, ok := m.graphs[id]
	if !ok {
		return nil, fmt.Errorf("graph %s not found", id)
	}
	return graph, nil
}

// Save stores a graph in the repository
func (m *DefaultGraphRepository) Save(graph *workflow.Graph) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.graphs[graph.ID()] = graph
	return nil
}
