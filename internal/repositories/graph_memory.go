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

	mu             sync.RWMutex
	graphs         map[string]*workflow.Graph
	versions       map[string][]workflow.SchemaVersion
	activeVersions map[string]int
}

// NewMemoryGraphRepository creates a new in-memory GraphRepository
func NewMemoryGraphRepository() GraphRepository {
	return &MemoryGraphRepository{
		graphs:         make(map[string]*workflow.Graph),
		versions:       make(map[string][]workflow.SchemaVersion),
		activeVersions: make(map[string]int),
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

// FindByIDAndVersion retrieves a graph for a specific schema version.
func (m *MemoryGraphRepository) FindByIDAndVersion(id string, version int) (*workflow.Graph, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	versions, ok := m.versions[id]
	if !ok {
		return nil, ErrSchemaVersionNotFound
	}

	for _, sv := range versions {
		if sv.Version == version {
			schema := sv.Schema.Clone()
			graph, err := workflow.NewGraph(&schema)
			if err != nil {
				return nil, err
			}
			return graph, nil
		}
	}

	return nil, ErrSchemaVersionNotFound
}

// SaveVersion persists a new SchemaVersion. If sv.IsActive is true the active-version
// pointer for this schema is updated to sv.Version.
func (m *MemoryGraphRepository) SaveVersion(sv *workflow.SchemaVersion) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sv.IsActive {
		for i := range m.versions[sv.SchemaID] {
			m.versions[sv.SchemaID][i].IsActive = false
		}
		m.activeVersions[sv.SchemaID] = sv.Version
	}

	m.versions[sv.SchemaID] = append(m.versions[sv.SchemaID], *sv)
	return nil
}

// ListVersions returns all recorded versions for a schema ordered by version ascending.
func (m *MemoryGraphRepository) ListVersions(schemaID string) ([]workflow.SchemaVersion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, ok := m.graphs[schemaID]; !ok {
		return nil, ErrGraphNotFound
	}

	versions := m.versions[schemaID]
	if len(versions) == 0 {
		return []workflow.SchemaVersion{}, nil
	}

	out := make([]workflow.SchemaVersion, len(versions))
	copy(out, versions)
	slices.SortFunc(out, func(a, b workflow.SchemaVersion) int {
		return a.Version - b.Version
	})
	return out, nil
}

// SetActiveVersion updates the active version pointer and rebuilds the cached graph from
// the stored version schema.
func (m *MemoryGraphRepository) SetActiveVersion(schemaID string, version int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	versions, ok := m.versions[schemaID]
	if !ok {
		return ErrGraphNotFound
	}

	found := false
	for i := range versions {
		if versions[i].Version == version {
			m.versions[schemaID][i].IsActive = true
			m.activeVersions[schemaID] = version
			found = true

			schema := versions[i].Schema.Clone()
			graph, err := workflow.NewGraph(&schema)
			if err != nil {
				return err
			}
			m.graphs[schemaID] = graph
		} else {
			m.versions[schemaID][i].IsActive = false
		}
	}

	if !found {
		return ErrSchemaVersionNotFound
	}

	return nil
}

// GetVersionHistory returns aggregate version metadata for a schema.
func (m *MemoryGraphRepository) GetVersionHistory(schemaID string) (*workflow.SchemaVersionHistory, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, ok := m.graphs[schemaID]; !ok {
		return nil, ErrGraphNotFound
	}

	versions := m.versions[schemaID]
	activeVersion := m.activeVersions[schemaID]
	latestVersion := 0
	for _, sv := range versions {
		if sv.Version > latestVersion {
			latestVersion = sv.Version
		}
	}

	return &workflow.SchemaVersionHistory{
		SchemaID:      schemaID,
		ActiveVersion: activeVersion,
		LatestVersion: latestVersion,
		TotalVersions: len(versions),
	}, nil
}
