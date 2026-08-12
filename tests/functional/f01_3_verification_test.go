package functional_test

// BACKLOG_V2.md F-01 finding 3 (BACKLOG_V2.md:53) — driver-parity half.
//
// Premise under test (verbatim): "graph_memory.go:FindByID returns the shared m.graphs[id] pointer
// live Workflow's hold, so updateVersioned -> graph.UpdateSchema rewrites a running run's topology
// in place."
//
// WHAT THIS PROVES: for each driver, whether a workflow run that is already in flight on schema v1
// keeps executing v1's topology after an ordinary schema edit (GraphService.Upsert) lands for the
// same schema id. The run is built exactly the way internal/actors/workflow_handler.go:167-177
// builds it — graphService.FindByID -> internalworkflow.New -> Trigger — and the edit arrives
// through the ordinary public path, so nothing here presumes where the corruption is introduced or
// where a fix would go.
//
// WHAT THIS DOES NOT PROVE: nothing about pointer identity, aliasing or copy semantics inside any
// repository — those are implementation choices, deliberately not asserted. Nothing about actor
// scheduling, process restart, or the durable half of the same problem (A-05). No ergo node is
// started; the run is driven through the workflow aggregate, which is the object the actor holds.
//
// The memory invocation below is untagged and runs under `make test`; the Postgres invocation lives
// in f01_3_verification_pg_test.go behind //go:build functional and needs DB_POSTGRES_DSN
// (tests/functional/postgres_test.go:25 skips silently without it).

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/mocks"
	"github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/internal/services"
	internalworkflow "github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/pkg/llm"
	pkgworkflow "github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Node ids inside mocks.SmallTestGraphSchema that this file manipulates.
const (
	f013TriggerNode = "debug-nil"
	f013BranchNode  = "logic-rand-2"
)

// f013LinearSchema returns mocks.SmallTestGraphSchema with the logic-rand-2 branch removed, so the
// two schemas share an ID but differ in topology: debug-nil forks into 2 nodes in v1 and into 1 in
// v2.
func f013LinearSchema() *internalworkflow.GraphSchema {
	schema := mocks.SmallTestGraphSchema()

	nodes := make([]*internalworkflow.NodeSchema, 0, len(schema.Nodes))
	for _, n := range schema.Nodes {
		if n.ID == f013BranchNode {
			continue
		}
		nodes = append(nodes, n)
	}
	schema.Nodes = nodes

	edges := make([]*internalworkflow.EdgeSchema, 0, len(schema.Edges))
	for _, e := range schema.Edges {
		if e.From == f013BranchNode || e.To == f013BranchNode {
			continue
		}
		edges = append(edges, e)
	}
	schema.Edges = edges

	return schema
}

// f013GraphService builds a GraphService over the driver's repository with the internal packages
// registered, matching how the server wires it (internal/app/di).
func f013GraphService(t *testing.T, repo repositories.GraphRepository) services.GraphService {
	t.Helper()

	pkgRegistry := packages.NewPackageRegistry()
	internalPackages := packages.NewInternal(llm.NewRegistry(nil, ""), pkgRegistry, nil)
	pkgSvc := services.NewPackageService(repositories.NewMemoryPackageRepository(), pkgRegistry, internalPackages)
	require.NoError(t, pkgSvc.RegisterInternalPackages())

	return services.NewGraphService(repo, pkgRegistry, nil)
}

// contractTestGraphLiveSchemaEditIsolation is the shared body both drivers are held to: a run that
// is already in flight must not observe a schema edit that lands after it started.
func contractTestGraphLiveSchemaEditIsolation(t *testing.T, newRepo func() repositories.GraphRepository, reset func()) {
	t.Helper()

	t.Run("an in-flight run keeps the topology it started on across a schema edit", func(t *testing.T) {
		// Arrange — schema v1 exists and a run is started on it.
		reset()
		svc := f013GraphService(t, newRepo())
		v1 := mocks.SmallTestGraphSchema()
		_, err := svc.Upsert(v1.ID, v1)
		require.NoError(t, err)

		graphRef, err := svc.FindByID(v1.ID)
		require.NoError(t, err)
		run := internalworkflow.New(pkgworkflow.NewID(), graphRef, "default")
		require.NotNil(t, run.Trigger(), "precondition: the run started")

		triggerNode, err := run.Graph().FindNode(f013TriggerNode)
		require.NoError(t, err)
		require.Len(t, triggerNode.OutputEdges(), 2, "precondition: the run started on the forking v1 topology")
		_, err = run.Graph().FindNode(f013BranchNode)
		require.NoError(t, err, "precondition: logic-rand-2 is part of v1")

		// Act — an ordinary schema edit for the same id lands while the run is in flight.
		_, err = svc.Upsert(v1.ID, f013LinearSchema())
		require.NoError(t, err)

		// Assert — the in-flight run still sees v1.
		_, err = run.Graph().FindNode(f013BranchNode)
		assert.NoError(t, err,
			"the in-flight run lost node logic-rand-2: a schema edit rewrote the topology of a run that had already started")

		triggerNodeAfter, err := run.Graph().FindNode(f013TriggerNode)
		if assert.NoError(t, err, "the in-flight run lost its trigger node debug-nil after a schema edit") {
			assert.Len(t, triggerNodeAfter.OutputEdges(), 2,
				"the in-flight run's fork degraded from 2 output edges to %d after a schema edit it should not see",
				len(triggerNodeAfter.OutputEdges()))
		}
	})
}

func TestMemoryGraphRepository_LiveSchemaEditIsolation_Contract(t *testing.T) {
	contractTestGraphLiveSchemaEditIsolation(t, repositories.NewMemoryGraphRepository, func() {})
}
