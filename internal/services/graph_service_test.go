package services_test

import (
	"encoding/json"
	"testing"

	"ergo.services/ergo/gen"

	"github.com/open-source-cloud/fuse/internal/mocks"
	"github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/internal/services"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/stretchr/testify/require"
)

type recordingSchemaPublisher struct {
	upserts int
}

func (r *recordingSchemaPublisher) PublishLocalUpsert(_ string, _ *workflow.GraphSchema) {
	r.upserts++
}

func (r *recordingSchemaPublisher) BindNode(gen.Node) {}

// TestGraphService tests the GraphService
func TestGraphService(t *testing.T) {
	memGraphRepo := repositories.NewMemoryGraphRepository()

	pkgRepo := repositories.NewMemoryPackageRepository()
	pkgRegistry := packages.NewPackageRegistry()
	internalPackages := packages.NewInternal()

	pkgSvc := services.NewPackageService(pkgRepo, pkgRegistry, internalPackages)
	if err := pkgSvc.RegisterInternalPackages(); err != nil {
		t.Fatalf("failed to register internal packages: %v", err)
	}

	graphService := services.NewGraphService(memGraphRepo, pkgRegistry, nil)

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

func TestGraphService_Upsert_invokesPublisher(t *testing.T) {
	memGraphRepo := repositories.NewMemoryGraphRepository()
	pkgRegistry := packages.NewPackageRegistry()
	internalPackages := packages.NewInternal()
	pkgSvc := services.NewPackageService(repositories.NewMemoryPackageRepository(), pkgRegistry, internalPackages)
	require.NoError(t, pkgSvc.RegisterInternalPackages())

	pub := &recordingSchemaPublisher{}
	graphService := services.NewGraphService(memGraphRepo, pkgRegistry, pub)

	schema := mocks.SmallTestGraphSchema()
	_, err := graphService.Upsert(schema.ID, schema)
	require.NoError(t, err)
	require.Equal(t, 1, pub.upserts)
}

func TestGraphService_Upsert_pathSchemaIDOverridesBodyID(t *testing.T) {
	memGraphRepo := repositories.NewMemoryGraphRepository()
	pkgRegistry := packages.NewPackageRegistry()
	internalPackages := packages.NewInternal()
	pkgSvc := services.NewPackageService(repositories.NewMemoryPackageRepository(), pkgRegistry, internalPackages)
	require.NoError(t, pkgSvc.RegisterInternalPackages())

	graphService := services.NewGraphService(memGraphRepo, pkgRegistry, nil)

	schema := mocks.SmallTestGraphSchema()
	schema.ID = "body-json-id"

	const apiID = "api-path-schema-id"
	_, err := graphService.Upsert(apiID, schema)
	require.NoError(t, err)

	g, err := graphService.FindByID(apiID)
	require.NoError(t, err)
	require.Equal(t, apiID, g.ID())
	require.Equal(t, apiID, g.Schema().ID)

	_, err = memGraphRepo.FindByID("body-json-id")
	require.ErrorIs(t, err, repositories.ErrGraphNotFound)
}

func TestGraphService_ApplyReplicatedUpsert(t *testing.T) {
	memGraphRepo := repositories.NewMemoryGraphRepository()
	pkgRegistry := packages.NewPackageRegistry()
	internalPackages := packages.NewInternal()
	pkgSvc := services.NewPackageService(repositories.NewMemoryPackageRepository(), pkgRegistry, internalPackages)
	require.NoError(t, pkgSvc.RegisterInternalPackages())

	pub := &recordingSchemaPublisher{}
	graphService := services.NewGraphService(memGraphRepo, pkgRegistry, pub)

	schema := mocks.SmallTestGraphSchema()
	payload, err := json.Marshal(schema)
	require.NoError(t, err)

	require.NoError(t, graphService.ApplyReplicatedUpsert(schema.ID, payload))
	require.Equal(t, 0, pub.upserts, "replicated apply must not publish")

	g, err := graphService.FindByID(schema.ID)
	require.NoError(t, err)
	require.Equal(t, schema.ID, g.ID())
}
