package repositories_test

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/tests"
)

// TestMemoryGraphRepository tests the MemoryGraphRepository
func TestMemoryGraphRepository(t *testing.T) {
	repo := repositories.NewMemoryGraphRepository()

	schema := tests.SmallTestGraphSchema()
	graph, err := workflow.NewGraph(schema)
	if err != nil {
		t.Fatalf("failed to create graph: %v", err)
	}

	err = repo.Save(graph)
	if err != nil {
		t.Fatalf("failed to save graph: %v", err)
	}

	existingGraph, err := repo.FindByID(graph.ID())
	if err != nil {
		t.Fatalf("failed to find graph: %v", err)
	}

	if existingGraph.ID() != graph.ID() {
		t.Fatalf("graph ID should be %s, got %s", graph.ID(), existingGraph.ID())
	}
}
