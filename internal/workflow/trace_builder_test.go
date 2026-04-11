package workflow

import (
	"testing"
	"time"

	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildTrace_LinearWorkflow(t *testing.T) {
	// Arrange
	now := time.Now()
	entries := []JournalEntry{
		{Sequence: 1, Timestamp: now, Type: JournalStateChanged, State: StateRunning},
		{Sequence: 2, Timestamp: now.Add(10 * time.Millisecond), Type: JournalStepStarted, ExecID: "exec-1", ThreadID: 0, FunctionNodeID: "node-1", Input: map[string]any{"url": "https://example.com"}},
		{Sequence: 3, Timestamp: now.Add(100 * time.Millisecond), Type: JournalStepCompleted, ExecID: "exec-1", Result: &workflow.FunctionResult{Output: workflow.FunctionOutput{Status: workflow.FunctionSuccess, Data: map[string]any{"status": 200}}}},
		{Sequence: 4, Timestamp: now.Add(110 * time.Millisecond), Type: JournalStepStarted, ExecID: "exec-2", ThreadID: 0, FunctionNodeID: "node-2"},
		{Sequence: 5, Timestamp: now.Add(200 * time.Millisecond), Type: JournalStepCompleted, ExecID: "exec-2", Result: &workflow.FunctionResult{Output: workflow.FunctionOutput{Status: workflow.FunctionSuccess, Data: map[string]any{"done": true}}}},
		{Sequence: 6, Timestamp: now.Add(210 * time.Millisecond), Type: JournalStateChanged, State: StateFinished},
	}

	// Act
	trace := BuildTrace("wf-1", "schema-1", entries)

	// Assert
	assert.Equal(t, "wf-1", trace.WorkflowID)
	assert.Equal(t, "schema-1", trace.SchemaID)
	assert.Equal(t, StateFinished, trace.Status)
	assert.Equal(t, now, trace.TriggeredAt)
	require.NotNil(t, trace.CompletedAt)
	require.NotNil(t, trace.Duration)
	assert.Nil(t, trace.Error)

	require.Len(t, trace.Steps, 2)

	step1 := trace.Steps[0]
	assert.Equal(t, "exec-1", step1.ExecID)
	assert.Equal(t, "node-1", step1.FunctionNodeID)
	assert.Equal(t, "completed", step1.Status)
	assert.Equal(t, 1, step1.Attempt)
	assert.Equal(t, map[string]any{"url": "https://example.com"}, step1.Input)
	require.NotNil(t, step1.Output)
	assert.Equal(t, map[string]any{"status": 200}, step1.Output.Data)
	require.NotNil(t, step1.Duration)

	step2 := trace.Steps[1]
	assert.Equal(t, "exec-2", step2.ExecID)
	assert.Equal(t, "completed", step2.Status)
}

func TestBuildTrace_FailedStep(t *testing.T) {
	now := time.Now()
	entries := []JournalEntry{
		{Sequence: 1, Timestamp: now, Type: JournalStateChanged, State: StateRunning},
		{Sequence: 2, Timestamp: now.Add(10 * time.Millisecond), Type: JournalStepStarted, ExecID: "exec-1", ThreadID: 0, FunctionNodeID: "node-1"},
		{Sequence: 3, Timestamp: now.Add(100 * time.Millisecond), Type: JournalStepFailed, ExecID: "exec-1", Result: &workflow.FunctionResult{Output: workflow.FunctionOutput{Status: workflow.FunctionError, Data: map[string]any{"error": "connection refused"}}}},
		{Sequence: 4, Timestamp: now.Add(110 * time.Millisecond), Type: JournalStateChanged, State: StateError},
	}

	trace := BuildTrace("wf-2", "schema-1", entries)

	assert.Equal(t, StateError, trace.Status)
	require.Len(t, trace.Steps, 1)
	assert.Equal(t, "failed", trace.Steps[0].Status)
	require.NotNil(t, trace.Steps[0].Error)
	assert.Equal(t, "connection refused", *trace.Steps[0].Error)
	require.NotNil(t, trace.Error)
	assert.Equal(t, "connection refused", *trace.Error)
}

func TestBuildTrace_RetryThenSuccess(t *testing.T) {
	now := time.Now()
	entries := []JournalEntry{
		{Sequence: 1, Timestamp: now, Type: JournalStateChanged, State: StateRunning},
		{Sequence: 2, Timestamp: now.Add(10 * time.Millisecond), Type: JournalStepStarted, ExecID: "exec-1", ThreadID: 0, FunctionNodeID: "node-1"},
		{Sequence: 3, Timestamp: now.Add(50 * time.Millisecond), Type: JournalStepRetrying, ExecID: "exec-1"},
		{Sequence: 4, Timestamp: now.Add(100 * time.Millisecond), Type: JournalStepRetrying, ExecID: "exec-1"},
		{Sequence: 5, Timestamp: now.Add(200 * time.Millisecond), Type: JournalStepCompleted, ExecID: "exec-1", Result: &workflow.FunctionResult{Output: workflow.FunctionOutput{Status: workflow.FunctionSuccess, Data: map[string]any{"ok": true}}}},
		{Sequence: 6, Timestamp: now.Add(210 * time.Millisecond), Type: JournalStateChanged, State: StateFinished},
	}

	trace := BuildTrace("wf-3", "schema-1", entries)

	assert.Equal(t, StateFinished, trace.Status)
	require.Len(t, trace.Steps, 1)
	assert.Equal(t, "completed", trace.Steps[0].Status)
	assert.Equal(t, 3, trace.Steps[0].Attempt) // 1 initial + 2 retries
}

func TestBuildTrace_ParallelThreads(t *testing.T) {
	now := time.Now()
	entries := []JournalEntry{
		{Sequence: 1, Timestamp: now, Type: JournalStateChanged, State: StateRunning},
		{Sequence: 2, Timestamp: now.Add(10 * time.Millisecond), Type: JournalStepStarted, ExecID: "exec-1", ThreadID: 1, FunctionNodeID: "node-a"},
		{Sequence: 3, Timestamp: now.Add(10 * time.Millisecond), Type: JournalStepStarted, ExecID: "exec-2", ThreadID: 2, FunctionNodeID: "node-b"},
		{Sequence: 4, Timestamp: now.Add(100 * time.Millisecond), Type: JournalStepCompleted, ExecID: "exec-1", Result: &workflow.FunctionResult{Output: workflow.FunctionOutput{Status: workflow.FunctionSuccess}}},
		{Sequence: 5, Timestamp: now.Add(150 * time.Millisecond), Type: JournalStepCompleted, ExecID: "exec-2", Result: &workflow.FunctionResult{Output: workflow.FunctionOutput{Status: workflow.FunctionSuccess}}},
		{Sequence: 6, Timestamp: now.Add(160 * time.Millisecond), Type: JournalStateChanged, State: StateFinished},
	}

	trace := BuildTrace("wf-4", "schema-1", entries)

	require.Len(t, trace.Steps, 2)
	assert.Equal(t, uint16(1), trace.Steps[0].ThreadID)
	assert.Equal(t, uint16(2), trace.Steps[1].ThreadID)
	assert.Equal(t, "completed", trace.Steps[0].Status)
	assert.Equal(t, "completed", trace.Steps[1].Status)
}

func TestBuildTrace_CancelledWorkflow(t *testing.T) {
	now := time.Now()
	entries := []JournalEntry{
		{Sequence: 1, Timestamp: now, Type: JournalStateChanged, State: StateRunning},
		{Sequence: 2, Timestamp: now.Add(10 * time.Millisecond), Type: JournalStepStarted, ExecID: "exec-1", ThreadID: 0, FunctionNodeID: "node-1"},
		{Sequence: 3, Timestamp: now.Add(500 * time.Millisecond), Type: JournalStateChanged, State: StateCancelled},
	}

	trace := BuildTrace("wf-5", "schema-1", entries)

	assert.Equal(t, StateCancelled, trace.Status)
	require.NotNil(t, trace.CompletedAt)
	require.Len(t, trace.Steps, 1)
	assert.Equal(t, "running", trace.Steps[0].Status) // step never completed
}

func TestBuildTrace_EmptyEntries(t *testing.T) {
	trace := BuildTrace("wf-6", "schema-1", nil)

	assert.Equal(t, "wf-6", trace.WorkflowID)
	assert.Equal(t, "schema-1", trace.SchemaID)
	assert.Empty(t, trace.Steps)
	assert.True(t, trace.TriggeredAt.IsZero())
}
