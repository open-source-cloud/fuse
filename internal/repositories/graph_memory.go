package repositories

import (
	"slices"
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
		return nil, ErrGraphNotFound
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

// List returns all stored graphs as lightweight list items, sorted by schema ID.
func (m *MemoryGraphRepository) List() ([]GraphSchemaListItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]GraphSchemaListItem, 0, len(m.graphs))
	for _, g := range m.graphs {
		s := g.Schema()
		out = append(out, GraphSchemaListItem{SchemaID: s.ID, Name: s.Name})
	}
	slices.SortFunc(out, func(a, b GraphSchemaListItem) int {
		if a.SchemaID < b.SchemaID {
			return -1
		}
		if a.SchemaID > b.SchemaID {
			return 1
		}
		return 0
	})
	return out, nil
}
