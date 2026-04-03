package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/open-source-cloud/fuse/internal/workflow/workflowactions"
	pkgwf "github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/require"
)

func TestHandleNodeFailure_RoutesOnErrorEdgeWhenRetriesExhausted(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "workflows", "error-edge-test.json"))
	require.NoError(t, err)

	schema, err := NewGraphSchemaFromJSON(raw)
	require.NoError(t, err)

	g, err := NewGraph(schema)
	require.NoError(t, err)

	w := New(pkgwf.ID("wf-error-edge"), g)

	failNode, err := g.FindNode("fail-node")
	require.NoError(t, err)

	threadID := failNode.Thread()
	execID := pkgwf.NewExecID(threadID)
	w.threads.New(threadID, execID)
	w.auditLog.NewEntry(threadID, failNode.ID(), execID.String(), map[string]any{"failCount": float64(999)})

	act := w.HandleNodeFailure(threadID, execID)
	run, ok := act.(*workflowactions.RunFunctionAction)
	require.True(t, ok, "expected RunFunctionAction, got %T", act)
	require.Equal(t, "fuse/pkg/debug/nil", run.FunctionID)

	recovery, err := g.FindNode("recovery")
	require.NoError(t, err)
	require.Equal(t, recovery.Thread(), run.ThreadID)
}

func TestHandleNodeFailure_RetriesWhenPolicyAllows(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "workflows", "parallel-retry-test.json"))
	require.NoError(t, err)

	schema, err := NewGraphSchemaFromJSON(raw)
	require.NoError(t, err)

	g, err := NewGraph(schema)
	require.NoError(t, err)

	w := New(pkgwf.ID("wf-parallel-retry"), g)

	branchB, err := g.FindNode("branch-b")
	require.NoError(t, err)

	threadID := branchB.Thread()
	execID := pkgwf.NewExecID(threadID)
	w.threads.New(threadID, execID)
	w.auditLog.NewEntry(threadID, branchB.ID(), execID.String(), map[string]any{"failCount": float64(1)})

	act1 := w.HandleNodeFailure(threadID, execID)
	_, ok := act1.(*workflowactions.RetryFunctionAction)
	require.True(t, ok, "first failure should schedule retry, got %T", act1)

	act2 := w.HandleNodeFailure(threadID, execID)
	_, ok = act2.(*workflowactions.RetryFunctionAction)
	require.True(t, ok, "second failure should schedule retry, got %T", act2)

	act3 := w.HandleNodeFailure(threadID, execID)
	require.Nil(t, act3, "after max retries with no onError edge, expect nil (terminal error)")
}
