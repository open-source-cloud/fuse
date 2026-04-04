package workflow

import (
	"testing"
	"time"

	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildExecutionSnapshot_SimpleWorkflow(t *testing.T) {
	// Arrange — a simple 1-thread, 1-node workflow
	now := time.Now().UTC()
	result := workflow.NewFunctionResultSuccessWith(map[string]any{"value": 42})
	entries := []JournalEntry{
		{Sequence: 1, Timestamp: now, Type: JournalStateChanged, State: StateRunning},
		{Sequence: 2, Timestamp: now, Type: JournalThreadCreated, ThreadID: 0},
		{Sequence: 3, Timestamp: now.Add(1 * time.Millisecond), Type: JournalStepStarted, ThreadID: 0, FunctionNodeID: "node-1", ExecID: "exec-1", Input: map[string]any{"x": 1}},
		{Sequence: 4, Timestamp: now.Add(10 * time.Millisecond), Type: JournalStepCompleted, ThreadID: 0, FunctionNodeID: "node-1", ExecID: "exec-1", Result: &result},
		{Sequence: 5, Timestamp: now.Add(11 * time.Millisecond), Type: JournalThreadDone, ThreadID: 0},
		{Sequence: 6, Timestamp: now.Add(12 * time.Millisecond), Type: JournalStateChanged, State: StateFinished},
	}
	outputs := map[string]any{"node-1": map[string]any{"value": 42}}

	// Act
	snap := BuildExecutionSnapshot("wf-1", "schema-1", StateFinished, entries, outputs)

	// Assert
	assert.Equal(t, 1, snap.SchemaVersion)
	assert.Equal(t, "wf-1", snap.WorkflowID)
	assert.Equal(t, "schema-1", snap.SchemaID)
	assert.Equal(t, "finished", snap.Status)
	require.NotNil(t, snap.StartedAt)
	require.NotNil(t, snap.FinishedAt)

	// Node runs
	require.Len(t, snap.NodeRuns, 1)
	run := snap.NodeRuns[0]
	assert.Equal(t, "exec-1", run.ExecID)
	assert.Equal(t, "node-1", run.NodeID)
	assert.Equal(t, "completed", run.Status)
	assert.Equal(t, map[string]any{"x": 1}, run.Input)
	assert.Equal(t, map[string]any{"value": 42}, run.Output)
	assert.Equal(t, uint64(3), run.JournalSequenceStart)
	assert.Equal(t, uint64(4), run.JournalSequenceEnd)

	// Threads
	require.Len(t, snap.Threads, 1)
	assert.Equal(t, "finished", snap.Threads[0].State)
	assert.Equal(t, "exec-1", snap.Threads[0].LastExecID)

	// Timeline
	assert.Len(t, snap.Timeline, 6)

	// Aggregated outputs
	assert.Equal(t, outputs, snap.AggregatedOutputs)
}

func TestBuildExecutionSnapshot_ParallelBranches(t *testing.T) {
	// Arrange — 2 parallel threads
	now := time.Now().UTC()
	r1 := workflow.NewFunctionResultSuccessWith(map[string]any{"a": 1})
	r2 := workflow.NewFunctionResultSuccessWith(map[string]any{"b": 2})
	entries := []JournalEntry{
		{Sequence: 1, Timestamp: now, Type: JournalStateChanged, State: StateRunning},
		{Sequence: 2, Timestamp: now, Type: JournalThreadCreated, ThreadID: 0},
		{Sequence: 3, Timestamp: now, Type: JournalThreadCreated, ThreadID: 1},
		{Sequence: 4, Timestamp: now.Add(1 * time.Millisecond), Type: JournalStepStarted, ThreadID: 0, FunctionNodeID: "node-a", ExecID: "exec-a"},
		{Sequence: 5, Timestamp: now.Add(1 * time.Millisecond), Type: JournalStepStarted, ThreadID: 1, FunctionNodeID: "node-b", ExecID: "exec-b"},
		{Sequence: 6, Timestamp: now.Add(5 * time.Millisecond), Type: JournalStepCompleted, ThreadID: 0, ExecID: "exec-a", Result: &r1},
		{Sequence: 7, Timestamp: now.Add(8 * time.Millisecond), Type: JournalStepCompleted, ThreadID: 1, ExecID: "exec-b", Result: &r2},
		{Sequence: 8, Timestamp: now.Add(9 * time.Millisecond), Type: JournalThreadDone, ThreadID: 0},
		{Sequence: 9, Timestamp: now.Add(9 * time.Millisecond), Type: JournalThreadDone, ThreadID: 1},
		{Sequence: 10, Timestamp: now.Add(10 * time.Millisecond), Type: JournalStateChanged, State: StateFinished},
	}

	// Act
	snap := BuildExecutionSnapshot("wf-2", "schema-2", StateFinished, entries, nil)

	// Assert
	require.Len(t, snap.NodeRuns, 2)
	assert.Equal(t, "completed", snap.NodeRuns[0].Status)
	assert.Equal(t, "completed", snap.NodeRuns[1].Status)
	require.Len(t, snap.Threads, 2)
	assert.Equal(t, "finished", snap.Status)
}

func TestBuildExecutionSnapshot_FailedWithRetry(t *testing.T) {
	// Arrange — node fails, retries, then succeeds
	now := time.Now().UTC()
	failResult := workflow.NewFunctionResultSuccessWith(map[string]any{"error": "timeout"})
	failResult.Output.Status = workflow.FunctionError
	successResult := workflow.NewFunctionResultSuccessWith(map[string]any{"ok": true})

	entries := []JournalEntry{
		{Sequence: 1, Timestamp: now, Type: JournalStateChanged, State: StateRunning},
		{Sequence: 2, Timestamp: now, Type: JournalThreadCreated, ThreadID: 0},
		{Sequence: 3, Timestamp: now.Add(1 * time.Millisecond), Type: JournalStepStarted, ThreadID: 0, FunctionNodeID: "node-1", ExecID: "exec-1"},
		{Sequence: 4, Timestamp: now.Add(5 * time.Millisecond), Type: JournalStepFailed, ThreadID: 0, ExecID: "exec-1", Result: &failResult},
		{Sequence: 5, Timestamp: now.Add(6 * time.Millisecond), Type: JournalStepRetrying, ThreadID: 0, ExecID: "exec-1"},
		{Sequence: 6, Timestamp: now.Add(10 * time.Millisecond), Type: JournalStepStarted, ThreadID: 0, FunctionNodeID: "node-1", ExecID: "exec-1"},
		{Sequence: 7, Timestamp: now.Add(15 * time.Millisecond), Type: JournalStepCompleted, ThreadID: 0, ExecID: "exec-1", Result: &successResult},
		{Sequence: 8, Timestamp: now.Add(16 * time.Millisecond), Type: JournalStateChanged, State: StateFinished},
	}

	// Act
	snap := BuildExecutionSnapshot("wf-3", "schema-3", StateFinished, entries, nil)

	// Assert — the retry is recorded, final status is completed
	require.Len(t, snap.NodeRuns, 2) // started twice = 2 node runs with same execID
	// First run: failed then retrying
	assert.Equal(t, "retrying", snap.NodeRuns[0].Status)
	assert.Equal(t, 1, snap.NodeRuns[0].RetryAttempt)
	// Second run: completed
	assert.Equal(t, "completed", snap.NodeRuns[1].Status)
	assert.Equal(t, "finished", snap.Status)
}

func TestBuildExecutionSnapshot_ErrorTerminal(t *testing.T) {
	// Arrange — workflow ends in error
	now := time.Now().UTC()
	failResult := workflow.NewFunctionResultSuccessWith(map[string]any{"error": "fatal error"})
	failResult.Output.Status = workflow.FunctionError

	entries := []JournalEntry{
		{Sequence: 1, Timestamp: now, Type: JournalStateChanged, State: StateRunning},
		{Sequence: 2, Timestamp: now, Type: JournalThreadCreated, ThreadID: 0},
		{Sequence: 3, Timestamp: now.Add(1 * time.Millisecond), Type: JournalStepStarted, ThreadID: 0, FunctionNodeID: "node-1", ExecID: "exec-1"},
		{Sequence: 4, Timestamp: now.Add(5 * time.Millisecond), Type: JournalStepFailed, ThreadID: 0, ExecID: "exec-1", Result: &failResult},
		{Sequence: 5, Timestamp: now.Add(6 * time.Millisecond), Type: JournalStateChanged, State: StateError},
	}

	// Act
	snap := BuildExecutionSnapshot("wf-4", "schema-4", StateError, entries, nil)

	// Assert
	assert.Equal(t, "error", snap.Status)
	require.NotNil(t, snap.Error)
	assert.Equal(t, "fatal error", *snap.Error)
	require.NotNil(t, snap.FinishedAt)
	require.Len(t, snap.NodeRuns, 1)
	assert.Equal(t, "failed", snap.NodeRuns[0].Status)
}

func TestBuildExecutionSnapshot_EmptyEntries(t *testing.T) {
	// Act
	snap := BuildExecutionSnapshot("wf-5", "schema-5", StateUntriggered, nil, nil)

	// Assert
	assert.Equal(t, 1, snap.SchemaVersion)
	assert.Equal(t, "untriggered", snap.Status)
	assert.Nil(t, snap.StartedAt)
	assert.Empty(t, snap.NodeRuns)
	assert.Empty(t, snap.Threads)
	assert.Empty(t, snap.Timeline)
}
