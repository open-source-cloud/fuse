package repositories

import (
	"fmt"
	"sync"

	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/rs/zerolog/log"
)

// MemoryGraphRepository is the default implementation of the GraphRepository interface (in-memory)
type MemoryGraphRepository struct {
	GraphRepository

	mu     sync.RWMutex
	graphs map[string]*workflow.Graph
}

// NewMemoryGraphRepository creates a new in-memory GraphRepository
func NewMemoryGraphRepository() GraphRepository {
	return &MemoryGraphRepository{
		graphs: make(map[string]*workflow.Graph),
	}
}

// FindByID retrieves a graph from the repository
func (m *MemoryGraphRepository) FindByID(id string) (*workflow.Graph, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	graph, ok := m.graphs[id]
	if !ok {
		return nil, fmt.Errorf("graph %s not found", id)
	}
	return graph, nil
}

// Save stores a graph in the repository
func (m *MemoryGraphRepository) Save(graph *workflow.Graph) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Info().Msgf("saving graph %s", graph.ID())

	m.graphs[graph.ID()] = graph

	log.Info().Msgf("graph saved %s", graph.ID())

	return nil
}
