package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/open-source-cloud/fuse/internal/workflow/workflowactions"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadTestGraph(t *testing.T) *Graph {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "workflows", "smallest-test.json"))
	require.NoError(t, err)
	schema, err := NewGraphSchemaFromJSON(raw)
	require.NoError(t, err)
	graph, err := NewGraph(schema)
	require.NoError(t, err)
	return graph
}

func setupFailedWorkflow(t *testing.T) (*Workflow, workflow.ExecID) {
	t.Helper()
	graph := loadTestGraph(t)
	wf := New(workflow.NewID(), graph)

	// Trigger the workflow
	action := wf.Trigger()
	require.NotNil(t, action)
	execID := action.(*workflowactions.RunFunctionAction).FunctionExecID

	// Simulate failure
	failResult := workflow.NewFunctionResultSuccessWith(map[string]any{"error": "test failure"})
	failResult.Output.Status = "error"
	wf.SetResultFor(execID, &failResult)

	// Set workflow to error state (as WorkflowHandler would do)
	wf.SetState(StateError)

	return wf, execID
}

func TestRetryNode_HappyPath(t *testing.T) {
	// Arrange
	wf, failedExecID := setupFailedWorkflow(t)

	// Act
	action, err := wf.RetryNode(failedExecID)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, action)

	runAction, ok := action.(*workflowactions.RunFunctionAction)
	require.True(t, ok, "should return RunFunctionAction")

	assert.NotEqual(t, failedExecID, runAction.FunctionExecID, "new exec ID should differ from failed")
	assert.Equal(t, StateRunning, wf.State(), "workflow should be back to running")

	// Verify journal entries were appended
	entries := wf.Journal().Entries()
	var hasManualRetry, hasStepStarted bool
	for _, e := range entries {
		if e.Type == JournalStepManualRetry && e.ExecID == runAction.FunctionExecID.String() {
			hasManualRetry = true
			assert.Equal(t, failedExecID.String(), e.Data["previousExecId"])
		}
		if e.Type == JournalStepStarted && e.ExecID == runAction.FunctionExecID.String() {
			hasStepStarted = true
		}
	}
	assert.True(t, hasManualRetry, "should have step:manual-retry entry")
	assert.True(t, hasStepStarted, "should have step:started entry for new exec")
}

func TestRetryNode_WrongState(t *testing.T) {
	// Arrange
	graph := loadTestGraph(t)
	wf := New(workflow.NewID(), graph)
	wf.SetState(StateRunning)

	// Act
	_, err := wf.RetryNode(workflow.NewExecID(0))

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected error")
}

func TestRetryNode_ExecNotFound(t *testing.T) {
	// Arrange
	wf, _ := setupFailedWorkflow(t)

	// Act
	_, err := wf.RetryNode(workflow.NewExecID(99))

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in audit log")
}

func TestRetryNode_ExecNotFailed(t *testing.T) {
	// Arrange — create a workflow with a completed (not failed) exec
	graph := loadTestGraph(t)
	wf := New(workflow.NewID(), graph)

	action := wf.Trigger()
	execID := action.(*workflowactions.RunFunctionAction).FunctionExecID

	// Complete successfully
	successResult := workflow.NewFunctionResultSuccessWith(map[string]any{"ok": true})
	wf.SetResultFor(execID, &successResult)
	wf.SetState(StateError) // Force error state for the precondition

	// Act
	_, err := wf.RetryNode(execID)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not have a step:failed")
}
