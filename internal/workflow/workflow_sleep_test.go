package workflow

import (
	"testing"
	"time"

	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJournal_SleepEntries(t *testing.T) {
	j := NewJournal()

	j.Append(JournalEntry{
		Type:     JournalSleepStarted,
		ThreadID: 0,
		ExecID:   "exec-1",
		Data:     map[string]any{"duration": "5s", "reason": "rate limit"},
	})
	j.Append(JournalEntry{
		Type:     JournalSleepCompleted,
		ThreadID: 0,
		ExecID:   "exec-1",
	})

	entries := j.Entries()
	require.Len(t, entries, 2)
	assert.Equal(t, JournalSleepStarted, entries[0].Type)
	assert.Equal(t, "5s", entries[0].Data["duration"])
	assert.Equal(t, "rate limit", entries[0].Data["reason"])
	assert.Equal(t, JournalSleepCompleted, entries[1].Type)
}

func TestJournal_AwakeableEntries(t *testing.T) {
	j := NewJournal()

	j.Append(JournalEntry{
		Type:     JournalAwakeableCreated,
		ThreadID: 1,
		ExecID:   "exec-2",
		Data:     map[string]any{"awakeableId": "awk-123", "timeout": "30s"},
	})
	j.Append(JournalEntry{
		Type:     JournalAwakeableResolved,
		ThreadID: 1,
		ExecID:   "exec-2",
		Data:     map[string]any{"awakeableId": "awk-123"},
	})

	entries := j.Entries()
	require.Len(t, entries, 2)
	assert.Equal(t, JournalAwakeableCreated, entries[0].Type)
	assert.Equal(t, "awk-123", entries[0].Data["awakeableId"])
	assert.Equal(t, JournalAwakeableResolved, entries[1].Type)
}

func TestJournal_SubWorkflowEntries(t *testing.T) {
	j := NewJournal()

	j.Append(JournalEntry{
		Type:     JournalSubWorkflowStarted,
		ThreadID: 0,
		ExecID:   "exec-3",
		Data: map[string]any{
			"childWorkflowId": "child-wf-1",
			"childSchemaId":   "child-schema",
			"async":           false,
		},
	})
	j.Append(JournalEntry{
		Type:     JournalSubWorkflowCompleted,
		ThreadID: 0,
		ExecID:   "exec-3",
		Data: map[string]any{
			"childWorkflowId": "child-wf-1",
			"childFinalState": "finished",
		},
	})

	entries := j.Entries()
	require.Len(t, entries, 2)
	assert.Equal(t, JournalSubWorkflowStarted, entries[0].Type)
	assert.Equal(t, "child-wf-1", entries[0].Data["childWorkflowId"])
	assert.Equal(t, JournalSubWorkflowCompleted, entries[1].Type)
}

func newMinimalWorkflow(t *testing.T) *Workflow {
	t.Helper()
	schema := &GraphSchema{
		ID:   "test-wf",
		Name: "test workflow",
		Nodes: []*NodeSchema{
			{ID: "n1", Function: "debug/nil"},
			{ID: "n2", Function: "debug/nil"},
		},
		Edges: []*EdgeSchema{
			{ID: "e1", From: "n1", To: "n2"},
		},
	}
	graph, err := NewGraph(schema)
	require.NoError(t, err)
	return New(workflow.NewID(), graph)
}

func TestReplayJournalEntries_SleepState(t *testing.T) {
	wf := newMinimalWorkflow(t)

	entries := []JournalEntry{
		{Sequence: 1, Type: JournalThreadCreated, ThreadID: 0, ExecID: "exec-1"},
		{Sequence: 2, Type: JournalStepStarted, ThreadID: 0, FunctionNodeID: "n1", ExecID: "exec-1"},
		{Sequence: 3, Type: JournalStateChanged, State: StateRunning},
		{Sequence: 4, Type: JournalStateChanged, State: StateSleeping},
	}

	wf.replayJournalEntries(entries)

	assert.Equal(t, StateSleeping, wf.State())
}

func TestReplayJournalEntries_CancelledState(t *testing.T) {
	wf := newMinimalWorkflow(t)

	entries := []JournalEntry{
		{Sequence: 1, Type: JournalThreadCreated, ThreadID: 0, ExecID: "exec-1"},
		{Sequence: 2, Type: JournalStateChanged, State: StateRunning},
		{Sequence: 3, Type: JournalStateChanged, State: StateCancelled},
	}

	wf.replayJournalEntries(entries)

	assert.Equal(t, StateCancelled, wf.State())
}

func TestAwakeable_Lifecycle(t *testing.T) {
	now := time.Now()
	timeout := 30 * time.Second

	awakeable := &Awakeable{
		ID:         "awk-test-1",
		WorkflowID: workflow.NewID(),
		ExecID:     workflow.NewExecID(0),
		ThreadID:   0,
		CreatedAt:  now,
		Timeout:    timeout,
		DeadlineAt: now.Add(timeout),
		Status:     AwakeablePending,
	}

	assert.Equal(t, AwakeablePending, awakeable.Status)
	assert.Equal(t, "awk-test-1", awakeable.ID)
	assert.WithinDuration(t, now.Add(timeout), awakeable.DeadlineAt, time.Second)

	// Simulate resolution
	awakeable.Status = AwakeableResolved
	awakeable.Result = map[string]any{"approved": true}
	assert.Equal(t, AwakeableResolved, awakeable.Status)
	assert.Equal(t, true, awakeable.Result["approved"])
}

func TestAwakeable_StatusConstants(t *testing.T) {
	assert.Equal(t, AwakeableStatus("pending"), AwakeablePending)
	assert.Equal(t, AwakeableStatus("resolved"), AwakeableResolved)
	assert.Equal(t, AwakeableStatus("timed_out"), AwakeableTimedOut)
	assert.Equal(t, AwakeableStatus("cancelled"), AwakeableCancelled)
}

func TestSubWorkflowRef_Fields(t *testing.T) {
	parentID := workflow.NewID()
	childID := workflow.NewID()
	execID := workflow.NewExecID(0)

	ref := &SubWorkflowRef{
		ParentWorkflowID: parentID,
		ParentThreadID:   0,
		ParentExecID:     execID,
		ChildWorkflowID:  childID,
		ChildSchemaID:    "child-schema-v1",
		Async:            false,
	}

	assert.Equal(t, parentID, ref.ParentWorkflowID)
	assert.Equal(t, childID, ref.ChildWorkflowID)
	assert.Equal(t, "child-schema-v1", ref.ChildSchemaID)
	assert.False(t, ref.Async)
}
