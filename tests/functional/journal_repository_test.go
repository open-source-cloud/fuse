package functional_test

import (
	"testing"
	"time"

	"github.com/open-source-cloud/fuse/internal/repositories"
	internalworkflow "github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func contractTestJournalRepository(t *testing.T, newRepo func() repositories.JournalRepository) {
	t.Helper()

	t.Run("Append and LoadAll returns entries in sequence order", func(t *testing.T) {
		repo := newRepo()
		wfID := workflow.NewID().String()
		entries := []internalworkflow.JournalEntry{
			{Sequence: 1, Type: internalworkflow.JournalThreadCreated, ThreadID: 0},
			{Sequence: 2, Type: internalworkflow.JournalStepStarted, ThreadID: 0, FunctionNodeID: "node-1", ExecID: "exec-1"},
			{Sequence: 3, Type: internalworkflow.JournalStepCompleted, ThreadID: 0, FunctionNodeID: "node-1", ExecID: "exec-1"},
		}

		err := repo.Append(wfID, entries...)
		require.NoError(t, err)

		loaded, err := repo.LoadAll(wfID)
		require.NoError(t, err)
		require.Len(t, loaded, 3)
		assert.Equal(t, uint64(1), loaded[0].Sequence)
		assert.Equal(t, uint64(2), loaded[1].Sequence)
		assert.Equal(t, uint64(3), loaded[2].Sequence)
		assert.Equal(t, internalworkflow.JournalThreadCreated, loaded[0].Type)
		assert.Equal(t, internalworkflow.JournalStepStarted, loaded[1].Type)
		assert.Equal(t, internalworkflow.JournalStepCompleted, loaded[2].Type)
	})

	t.Run("LoadAll returns empty for nonexistent workflow", func(t *testing.T) {
		repo := newRepo()
		loaded, err := repo.LoadAll("nonexistent-wf")
		require.NoError(t, err)
		assert.Empty(t, loaded)
	})

	t.Run("LoadAll sorts by sequence even if appended out of order", func(t *testing.T) {
		repo := newRepo()
		wfID := workflow.NewID().String()
		require.NoError(t, repo.Append(wfID, internalworkflow.JournalEntry{Sequence: 3, Type: internalworkflow.JournalStepCompleted}))
		require.NoError(t, repo.Append(wfID, internalworkflow.JournalEntry{Sequence: 1, Type: internalworkflow.JournalThreadCreated}))
		require.NoError(t, repo.Append(wfID, internalworkflow.JournalEntry{Sequence: 2, Type: internalworkflow.JournalStepStarted}))

		loaded, err := repo.LoadAll(wfID)
		require.NoError(t, err)
		require.Len(t, loaded, 3)
		assert.Equal(t, uint64(1), loaded[0].Sequence)
		assert.Equal(t, uint64(2), loaded[1].Sequence)
		assert.Equal(t, uint64(3), loaded[2].Sequence)
	})

	t.Run("LastSequence returns 0 for empty journal", func(t *testing.T) {
		repo := newRepo()
		seq, err := repo.LastSequence("nonexistent-wf")
		require.NoError(t, err)
		assert.Equal(t, uint64(0), seq)
	})

	t.Run("LastSequence returns highest sequence", func(t *testing.T) {
		repo := newRepo()
		wfID := workflow.NewID().String()
		require.NoError(t, repo.Append(wfID,
			internalworkflow.JournalEntry{Sequence: 1},
			internalworkflow.JournalEntry{Sequence: 5},
			internalworkflow.JournalEntry{Sequence: 3},
		))

		seq, err := repo.LastSequence(wfID)
		require.NoError(t, err)
		assert.Equal(t, uint64(5), seq)
	})

	t.Run("Isolates entries between workflows", func(t *testing.T) {
		repo := newRepo()
		wf1 := workflow.NewID().String()
		wf2 := workflow.NewID().String()
		require.NoError(t, repo.Append(wf1, internalworkflow.JournalEntry{Sequence: 1, Type: internalworkflow.JournalThreadCreated}))
		require.NoError(t, repo.Append(wf2, internalworkflow.JournalEntry{Sequence: 1, Type: internalworkflow.JournalStepStarted}))

		entries1, err := repo.LoadAll(wf1)
		require.NoError(t, err)
		entries2, err := repo.LoadAll(wf2)
		require.NoError(t, err)

		require.Len(t, entries1, 1)
		assert.Equal(t, internalworkflow.JournalThreadCreated, entries1[0].Type)
		require.Len(t, entries2, 1)
		assert.Equal(t, internalworkflow.JournalStepStarted, entries2[0].Type)
	})

	t.Run("Preserves entry fields through round-trip", func(t *testing.T) {
		repo := newRepo()
		wfID := workflow.NewID().String()
		result := workflow.NewFunctionResultSuccessWith(map[string]any{"value": 42})
		entry := internalworkflow.JournalEntry{
			Sequence:       1,
			Timestamp:      time.Now().UTC().Truncate(time.Microsecond),
			Type:           internalworkflow.JournalStepCompleted,
			ThreadID:       5,
			FunctionNodeID: "my-node",
			ExecID:         workflow.NewExecID(5).String(),
			Input:          map[string]any{"key": "val"},
			Result:         &result,
		}

		require.NoError(t, repo.Append(wfID, entry))
		loaded, err := repo.LoadAll(wfID)
		require.NoError(t, err)
		require.Len(t, loaded, 1)

		e := loaded[0]
		assert.Equal(t, uint64(1), e.Sequence)
		assert.Equal(t, internalworkflow.JournalStepCompleted, e.Type)
		assert.Equal(t, uint16(5), e.ThreadID)
		assert.Equal(t, "my-node", e.FunctionNodeID)
		assert.Equal(t, "val", e.Input["key"])
		require.NotNil(t, e.Result)
	})

	t.Run("Handles entries with state changes", func(t *testing.T) {
		repo := newRepo()
		wfID := workflow.NewID().String()
		entry := internalworkflow.JournalEntry{
			Sequence: 1,
			Type:     internalworkflow.JournalStateChanged,
			State:    internalworkflow.StateRunning,
		}

		require.NoError(t, repo.Append(wfID, entry))
		loaded, err := repo.LoadAll(wfID)
		require.NoError(t, err)
		require.Len(t, loaded, 1)
		assert.Equal(t, internalworkflow.JournalStateChanged, loaded[0].Type)
		assert.Equal(t, internalworkflow.StateRunning, loaded[0].State)
	})

	t.Run("Handles entries with parent threads", func(t *testing.T) {
		repo := newRepo()
		wfID := workflow.NewID().String()
		entry := internalworkflow.JournalEntry{
			Sequence:      1,
			Type:          internalworkflow.JournalThreadCreated,
			ThreadID:      2,
			ParentThreads: []uint16{0, 1},
		}

		require.NoError(t, repo.Append(wfID, entry))
		loaded, err := repo.LoadAll(wfID)
		require.NoError(t, err)
		require.Len(t, loaded, 1)
		assert.Equal(t, []uint16{0, 1}, loaded[0].ParentThreads)
	})

	t.Run("Handles entries with data map", func(t *testing.T) {
		repo := newRepo()
		wfID := workflow.NewID().String()
		entry := internalworkflow.JournalEntry{
			Sequence: 1,
			Type:     internalworkflow.JournalSleepStarted,
			Data:     map[string]any{"duration": "5s", "reason": "backoff"},
		}

		require.NoError(t, repo.Append(wfID, entry))
		loaded, err := repo.LoadAll(wfID)
		require.NoError(t, err)
		require.Len(t, loaded, 1)
		assert.Equal(t, "5s", loaded[0].Data["duration"])
		assert.Equal(t, "backoff", loaded[0].Data["reason"])
	})
}

func TestMemoryJournalRepository_Contract(t *testing.T) {
	contractTestJournalRepository(t, repositories.NewMemoryJournalRepository)
}
