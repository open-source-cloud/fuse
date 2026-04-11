package workflow

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/internal/workflow/workflowactions"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildForEachGraph constructs a minimal graph:
//
//	trigger → foreach → {each: body, done: aggregate}
//	body → (no output edges)
func buildForEachGraph(t *testing.T) *Graph {
	t.Helper()
	schema := &GraphSchema{
		ID:   "foreach-test",
		Name: "foreach test",
		Nodes: []*NodeSchema{
			{ID: "trigger", Function: "debug/nil"},
			{ID: "foreach1", Function: "system/foreach"},
			{ID: "body", Function: "debug/nil"},
			{ID: "aggregate", Function: "debug/nil"},
		},
		Edges: []*EdgeSchema{
			{ID: "e-trigger-foreach", From: "trigger", To: "foreach1"},
			{
				ID:   "e-each",
				From: "foreach1",
				To:   "body",
				Conditional: &EdgeCondition{
					Name:  "each",
					Type:  ConditionExact,
					Value: "each",
				},
			},
			{
				ID:   "e-done",
				From: "foreach1",
				To:   "aggregate",
				Conditional: &EdgeCondition{
					Name:  "done",
					Type:  ConditionExact,
					Value: "done",
				},
			},
		},
	}
	g, err := NewGraph(schema)
	require.NoError(t, err)

	// Attach metadata so IsConditional() works for foreach1
	err = g.UpdateNodeMetadata("foreach1", &packages.FunctionMetadata{
		Output: packages.FunctionOutputMetadata{
			ConditionalOutput:      true,
			ConditionalOutputField: "_foreach_phase",
		},
	})
	require.NoError(t, err)
	err = g.UpdateNodeMetadata("trigger", &packages.FunctionMetadata{})
	require.NoError(t, err)
	err = g.UpdateNodeMetadata("body", &packages.FunctionMetadata{})
	require.NoError(t, err)
	err = g.UpdateNodeMetadata("aggregate", &packages.FunctionMetadata{})
	require.NoError(t, err)

	return g
}

// --- StartForEachIteration ---

func TestStartForEachIteration_ReturnsActionForBodyNode(t *testing.T) {
	g := buildForEachGraph(t)
	wf := New(workflow.NewID(), g)

	input := map[string]any{"item": "hello", "index": 0, "total": 1, "isLast": true}
	action, threadID, err := wf.StartForEachIteration("foreach1", input)

	require.NoError(t, err)
	assert.NotNil(t, action)
	assert.Equal(t, "debug/nil", action.FunctionID) // body node
	assert.Equal(t, threadID, action.ThreadID)
	assert.Equal(t, input, action.Args)
}

func TestStartForEachIteration_AllocatesUniqueThreads(t *testing.T) {
	g := buildForEachGraph(t)
	wf := New(workflow.NewID(), g)

	_, threadID1, err := wf.StartForEachIteration("foreach1", map[string]any{"index": 0})
	require.NoError(t, err)

	_, threadID2, err := wf.StartForEachIteration("foreach1", map[string]any{"index": 1})
	require.NoError(t, err)

	assert.NotEqual(t, threadID1, threadID2)
}

func TestStartForEachIteration_WritesJournalEntries(t *testing.T) {
	g := buildForEachGraph(t)
	wf := New(workflow.NewID(), g)

	_, threadID, err := wf.StartForEachIteration("foreach1", map[string]any{"item": "x"})
	require.NoError(t, err)

	entries := wf.Journal().Entries()
	// Should have: foreach:iteration:started + step:started
	var iterStarted, stepStarted bool
	for _, e := range entries {
		switch e.Type {
		case JournalForEachIterationStarted:
			iterStarted = true
			assert.Equal(t, threadID, e.ThreadID)
		case JournalStepStarted:
			stepStarted = true
			assert.Equal(t, threadID, e.ThreadID)
			assert.Equal(t, "body", e.FunctionNodeID)
		}
	}
	assert.True(t, iterStarted)
	assert.True(t, stepStarted)
}

func TestStartForEachIteration_ErrorOnMissingNode(t *testing.T) {
	g := buildForEachGraph(t)
	wf := New(workflow.NewID(), g)

	_, _, err := wf.StartForEachIteration("does-not-exist", nil)
	assert.Error(t, err)
}

func TestStartForEachIteration_ErrorOnMissingEachEdge(t *testing.T) {
	// Build a graph where the foreach node has only a "done" edge (no "each" edge).
	schema := &GraphSchema{
		ID:   "no-each-edge",
		Name: "test",
		Nodes: []*NodeSchema{
			{ID: "n1", Function: "debug/nil"},
			{ID: "foreach1", Function: "system/foreach"},
			{ID: "aggregate", Function: "debug/nil"},
		},
		Edges: []*EdgeSchema{
			{ID: "e1", From: "n1", To: "foreach1"},
			{
				ID:   "e-done",
				From: "foreach1",
				To:   "aggregate",
				Conditional: &EdgeCondition{
					Name:  "done",
					Type:  ConditionExact,
					Value: "done",
				},
			},
		},
	}
	g, err := NewGraph(schema)
	require.NoError(t, err)

	wf := New(workflow.NewID(), g)
	_, _, err = wf.StartForEachIteration("foreach1", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "each")
}

// --- CompleteForEach ---

func TestCompleteForEach_SetsResultAndPhase(t *testing.T) {
	g := buildForEachGraph(t)
	wf := New(workflow.NewID(), g)

	// Simulate trigger then foreach node being executed so there's an audit log entry.
	triggerExecID := workflow.NewExecID(0)
	wf.threads.New(0, triggerExecID)
	wf.auditLog.NewEntry(0, "trigger", triggerExecID.String(), nil)

	forEachExecID := workflow.NewExecID(0)
	wf.threads.New(0, forEachExecID)
	wf.auditLog.NewEntry(0, "foreach1", forEachExecID.String(), nil)

	results := []any{"r0", "r1", "r2"}
	wf.CompleteForEach(forEachExecID, results)

	// The aggregated output for foreach1 should contain _foreach_phase = "done"
	phase := wf.aggregatedOutput.Get("foreach1._foreach_phase")
	assert.Equal(t, "done", phase)

	resultSlice := wf.aggregatedOutput.Get("foreach1.results")
	assert.Equal(t, results, resultSlice)
}

// --- LastResultForThread ---

func TestLastResultForThread_ReturnsOutputData(t *testing.T) {
	g := buildForEachGraph(t)
	wf := New(workflow.NewID(), g)

	execID := workflow.NewExecID(5)
	wf.threads.New(5, execID)
	wf.auditLog.NewEntry(5, "body", execID.String(), nil)

	data := map[string]any{"processed": true}
	wf.SetResultFor(execID, &workflow.FunctionResult{
		Output: workflow.NewFunctionSuccessOutput(data),
	})

	result := wf.LastResultForThread(5)
	assert.Equal(t, data, result)
}

func TestLastResultForThread_UnknownThread_ReturnsNil(t *testing.T) {
	g := buildForEachGraph(t)
	wf := New(workflow.NewID(), g)
	assert.Nil(t, wf.LastResultForThread(99))
}

func TestLastResultForThread_NoResult_ReturnsNil(t *testing.T) {
	g := buildForEachGraph(t)
	wf := New(workflow.NewID(), g)

	execID := workflow.NewExecID(7)
	wf.threads.New(7, execID)
	wf.auditLog.NewEntry(7, "body", execID.String(), nil)
	// No SetResultFor call

	assert.Nil(t, wf.LastResultForThread(7))
}

// --- findNamedOutputEdge ---

func TestFindNamedOutputEdge_Found(t *testing.T) {
	g := buildForEachGraph(t)
	node, err := g.FindNode("foreach1")
	require.NoError(t, err)

	edge := findNamedOutputEdge(node, "each")
	require.NotNil(t, edge)
	assert.Equal(t, "body", edge.To().ID())
}

func TestFindNamedOutputEdge_NotFound(t *testing.T) {
	g := buildForEachGraph(t)
	node, err := g.FindNode("foreach1")
	require.NoError(t, err)

	edge := findNamedOutputEdge(node, "nonexistent")
	assert.Nil(t, edge)
}

// --- integration: full foreach iteration cycle ---

func TestForEach_FullCycleRoutes_ToDoneEdge(t *testing.T) {
	g := buildForEachGraph(t)
	wf := New(workflow.NewID(), g)

	// Simulate: trigger completes, then foreach is about to execute.
	triggerExecID := workflow.NewExecID(0)
	triggerThread := wf.threads.New(0, triggerExecID)
	wf.auditLog.NewEntry(0, "trigger", triggerExecID.String(), nil)
	triggerResult := map[string]any{"items": []any{"a", "b"}}
	wf.SetResultFor(triggerExecID, &workflow.FunctionResult{
		Output: workflow.NewFunctionSuccessOutput(triggerResult),
	})
	triggerThread.SetCurrentExecID(triggerExecID)

	// foreach node starts on the same thread (thread 0 inherits from trigger).
	forEachExecID := workflow.NewExecID(0)
	triggerThread.SetCurrentExecID(forEachExecID)
	wf.auditLog.NewEntry(0, "foreach1", forEachExecID.String(), map[string]any{"items": []any{"a", "b"}})

	// Spawn iteration threads.
	action0, tid0, err := wf.StartForEachIteration("foreach1", map[string]any{"item": "a", "index": 0})
	require.NoError(t, err)
	require.NotNil(t, action0)

	action1, tid1, err := wf.StartForEachIteration("foreach1", map[string]any{"item": "b", "index": 1})
	require.NoError(t, err)
	require.NotNil(t, action1)

	assert.NotEqual(t, tid0, tid1)

	// Simulate iteration 0 completing.
	wf.SetResultFor(action0.FunctionExecID, &workflow.FunctionResult{
		Output: workflow.NewFunctionSuccessOutput(map[string]any{"out": "A"}),
	})
	// thread 0's iteration: body has no output edges → Next returns Noop in reality,
	// but for this test we just verify CompleteForEach + Next routing.

	// Simulate iteration 1 completing.
	wf.SetResultFor(action1.FunctionExecID, &workflow.FunctionResult{
		Output: workflow.NewFunctionSuccessOutput(map[string]any{"out": "B"}),
	})

	// Mark both iteration threads finished (simulating Noop from Next).
	it0 := wf.threads.Get(tid0)
	require.NotNil(t, it0)
	it0.SetState(ThreadFinished)

	it1 := wf.threads.Get(tid1)
	require.NotNil(t, it1)
	it1.SetState(ThreadFinished)

	// All done — complete the foreach node.
	results := []any{map[string]any{"out": "A"}, map[string]any{"out": "B"}}
	wf.CompleteForEach(forEachExecID, results)

	// Calling Next on thread 0 should now follow the "done" edge to "aggregate".
	nextAction := wf.Next(0)
	require.NotNil(t, nextAction)
	assert.Equal(t, "function:run", string(nextAction.Type()))
	runAction, ok := nextAction.(*workflowactions.RunFunctionAction)
	require.True(t, ok)
	assert.Equal(t, "debug/nil", runAction.FunctionID) // aggregate node
}
