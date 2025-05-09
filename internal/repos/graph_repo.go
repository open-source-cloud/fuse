package repos

import (
	"fmt"
	"github.com/open-source-cloud/fuse/internal/workflow"
)

func NewGraphRepo() GraphRepo {
	return &memoryGraphRepo{
		graphs: make(map[string]*workflow.Graph),
	}
}

// GraphRepo workflow schema repository
type (
	GraphRepo interface {
		Get(id string) (*workflow.Graph, error)
		Save(graph *workflow.Graph) error
	}

	memoryGraphRepo struct {
		graphs map[string]*workflow.Graph
	}
)

func (m *memoryGraphRepo) Get(id string) (*workflow.Graph, error) {
	foundGraph, ok := m.graphs[id]
	if !ok {
		return nil, fmt.Errorf("graph %s not found", id)
	}

	return foundGraph, nil
}

func (m *memoryGraphRepo) Save(graph *workflow.Graph) error {
	m.graphs[graph.ID()] = graph
	return nil
}
