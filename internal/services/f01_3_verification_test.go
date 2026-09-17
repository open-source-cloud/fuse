package services_test

// BACKLOG_V2.md F-01 finding 3 (BACKLOG_V2.md:53) verification — memory driver half.
//
// Premise under test (verbatim): "graph_memory.go:FindByID returns the shared m.graphs[id] pointer
// live Workflow's hold, so updateVersioned -> graph.UpdateSchema rewrites a running run's topology
// in place."
//
// WHAT THIS PROVES: with the in-memory GraphRepository wired into DefaultGraphService, a run built
// the way internal/actors/workflow_handler.go:167-177 builds it (graphService.FindByID ->
// internalworkflow.New -> Trigger) has its topology rewritten underneath it by a later
// GraphService.Upsert of a different schema for the same id — it loses a node, loses a parallel
// branch at the next scheduling decision, and panics in Workflow.Next when the node it is executing
// is the one that disappeared.
//
// WHAT THIS DOES NOT PROVE: nothing about pointer identity or copy semantics anywhere — no test
// here asserts that any two values are or are not the same pointer, because that is the shape of a
// fix, not the shape of the defect. Nothing about the Postgres driver (see
// tests/functional/f01_3_verification_test.go for the parity contract), nothing about actor
// scheduling, and nothing about process restart. No ergo node is started here; the run is driven
// through the workflow aggregate directly, which is the same object the actor holds.

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/mocks"
	"github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/internal/services"
	internalworkflow "github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/internal/workflow/workflowactions"
	"github.com/open-source-cloud/fuse/pkg/llm"
	pkgworkflow "github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newF013GraphService builds a GraphService backed by the in-memory graph repository with the
// internal packages registered, so node metadata population succeeds.
func newF013GraphService(t *testing.T) services.GraphService {
	t.Helper()

	memGraphRepo := repositories.NewMemoryGraphRepository()
	pkgRegistry := packages.NewPackageRegistry()
	internalPackages := packages.NewInternal(llm.NewRegistry(nil, ""), pkgRegistry, nil)
	pkgSvc := services.NewPackageService(repositories.NewMemoryPackageRepository(), pkgRegistry, internalPackages)
	require.NoError(t, pkgSvc.RegisterInternalPackages())

	return services.NewGraphService(memGraphRepo, pkgRegistry, nil)
}

// Node ids inside mocks.SmallTestGraphSchema that this file manipulates.
const (
	f013TriggerNode        = "debug-nil"
	f013BranchNode         = "logic-rand-2"
	f013RenamedTriggerNode = "debug-nil-renamed"
)

// f013LinearSchema returns the same schema id as mocks.SmallTestGraphSchema but with the
// logic-rand-2 branch removed, so debug-nil forks into 2 nodes in v1 and into 1 in v2.
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

// TestGraphService_Upsert_MustNotMutateRunningWorkflowGraph asserts the specific corruption: a
// workflow that already triggered on schema v1 observes v2's node set without being restarted.
func TestGraphService_Upsert_MustNotMutateRunningWorkflowGraph(t *testing.T) {
	// Arrange
	svc := newF013GraphService(t)
	v1 := mocks.SmallTestGraphSchema()
	_, err := svc.Upsert(v1.ID, v1)
	require.NoError(t, err)

	// Mirrors internal/actors/workflow_handler.go:167-177 on the create path.
	graphRef, err := svc.FindByID(v1.ID)
	require.NoError(t, err)
	run := internalworkflow.New(pkgworkflow.NewID(), graphRef, "default")
	require.NotNil(t, run.Trigger(), "precondition: the run started")

	triggerNode, err := run.Graph().FindNode(f013TriggerNode)
	require.NoError(t, err)
	require.Len(t, triggerNode.OutputEdges(), 2, "precondition: the run started on the forking v1 topology")
	_, err = run.Graph().FindNode(f013BranchNode)
	require.NoError(t, err, "precondition: logic-rand-2 exists in v1")

	// Act — a schema edit arrives through the ordinary public path while the run is in flight.
	_, err = svc.Upsert(v1.ID, f013LinearSchema())
	require.NoError(t, err)

	// Assert
	_, err = run.Graph().FindNode(f013BranchNode)
	assert.NoError(t, err,
		"the in-flight run's graph lost node logic-rand-2: updateVersioned -> Graph.UpdateSchema rewrote the running run's topology in place")

	triggerNodeAfter, err := run.Graph().FindNode(f013TriggerNode)
	require.NoError(t, err)
	assert.Len(t, triggerNodeAfter.OutputEdges(), 2,
		"the in-flight run's fork degraded from 2 output edges to %d after an unrelated schema upsert", len(triggerNodeAfter.OutputEdges()))
}

// f013RenamedTriggerSchema returns the same schema id with the trigger node renamed, so a run that
// is in flight on "debug-nil" finds no such node after the edit.
func f013RenamedTriggerSchema() *internalworkflow.GraphSchema {
	schema := f013LinearSchema()
	for _, n := range schema.Nodes {
		if n.ID == f013TriggerNode {
			n.ID = f013RenamedTriggerNode
		}
	}
	for _, e := range schema.Edges {
		if e.From == f013TriggerNode {
			e.From = f013RenamedTriggerNode
		}
		if e.To == f013TriggerNode {
			e.To = f013RenamedTriggerNode
		}
	}
	return schema
}

// TestWorkflow_Next_AfterLiveSchemaEditDropsInFlightNode_MustNotPanic asserts the worst consequence of
// the shared pointer: Workflow.Next ignores the error from graph.FindNode
// (internal/workflow/workflow.go:312) and dereferences the nil node on the next line.
func TestWorkflow_Next_AfterLiveSchemaEditDropsInFlightNode_MustNotPanic(t *testing.T) {
	// Arrange
	svc := newF013GraphService(t)
	v1 := mocks.SmallTestGraphSchema()
	_, err := svc.Upsert(v1.ID, v1)
	require.NoError(t, err)

	graphRef, err := svc.FindByID(v1.ID)
	require.NoError(t, err)
	run := internalworkflow.New(pkgworkflow.NewID(), graphRef, "default")
	trigger, ok := run.Trigger().(*workflowactions.RunFunctionAction)
	require.True(t, ok)

	// Act
	_, err = svc.Upsert(v1.ID, f013RenamedTriggerSchema())
	require.NoError(t, err)

	// Assert
	assert.NotPanics(t, func() { _ = run.Next(trigger.ThreadID) },
		"the node the run is executing was deleted from its graph in place, and Next dereferences the nil lookup result")
}

// TestWorkflow_Next_AfterLiveSchemaEdit_MustKeepParallelBranch asserts the behavioural consequence:
// the same Next() call that yields a 2-way parallel action on an isolated v1 graph yields a
// single-function action on the run whose graph the repository handed out by pointer.
func TestWorkflow_Next_AfterLiveSchemaEdit_MustKeepParallelBranch(t *testing.T) {
	// Arrange
	svc := newF013GraphService(t)
	v1 := mocks.SmallTestGraphSchema()
	_, err := svc.Upsert(v1.ID, v1)
	require.NoError(t, err)

	victimGraph, err := svc.FindByID(v1.ID)
	require.NoError(t, err)
	victim := internalworkflow.New(pkgworkflow.NewID(), victimGraph, "default")
	victimTrigger, ok := victim.Trigger().(*workflowactions.RunFunctionAction)
	require.True(t, ok)

	controlGraph, err := internalworkflow.NewGraph(mocks.SmallTestGraphSchema())
	require.NoError(t, err)
	control := internalworkflow.New(pkgworkflow.NewID(), controlGraph, "default")
	controlTrigger, ok := control.Trigger().(*workflowactions.RunFunctionAction)
	require.True(t, ok)

	// Act — upsert the linear v2 schema; only the repository-held graph should be affected.
	_, err = svc.Upsert(v1.ID, f013LinearSchema())
	require.NoError(t, err)

	controlNext := control.Next(controlTrigger.ThreadID)
	victimNext := victim.Next(victimTrigger.ThreadID)

	// Assert
	require.Equal(t, workflowactions.ActionRunParallelFunctions, controlNext.Type(),
		"control run (graph never handed out by the repository) still forks")
	assert.Equal(t, workflowactions.ActionRunParallelFunctions, victimNext.Type(),
		"in-flight run scheduled %s instead of a 2-way parallel fork after a concurrent schema upsert", victimNext.Type())
}
