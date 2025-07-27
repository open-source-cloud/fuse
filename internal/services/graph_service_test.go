package services_test

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/mocks"
	"github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/internal/services"
)

// TestGraphService tests the GraphService
func TestGraphService(t *testing.T) {
	memGraphRepo := repositories.NewMemoryGraphRepository()

	pkgRegistry := packages.NewPackageRegistry()
	internalPackages := packages.NewInternal(pkgRegistry)
	internalPackages.Register()

	graphService := services.NewGraphService(memGraphRepo, pkgRegistry)

	schema := mocks.SmallTestGraphSchema()

	graph, err := graphService.Upsert(schema.ID, schema)
	if err != nil {
		t.Fatalf("failed to upsert graph: %v", err)
	}

	if graph.ID() != schema.ID {
		t.Fatalf("graph ID should be %s, got %s", schema.ID, graph.ID())
	}

	existingGraph, err := graphService.FindByID(graph.ID())
	if err != nil {
		t.Fatalf("failed to find graph: %v", err)
	}

	if existingGraph.ID() != graph.ID() {
		t.Fatalf("graph ID should be %s, got %s", graph.ID(), existingGraph.ID())
	}
}
