package repositories_test

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryJournalRepository_Append_And_LoadAll(t *testing.T) {
	// Arrange
	repo := repositories.NewMemoryJournalRepository()
	entries := []workflow.JournalEntry{
		{Sequence: 1, Type: workflow.JournalThreadCreated, ThreadID: 0},
		{Sequence: 2, Type: workflow.JournalStepStarted, ThreadID: 0, FunctionNodeID: "node-1"},
		{Sequence: 3, Type: workflow.JournalStepCompleted, ThreadID: 0, FunctionNodeID: "node-1"},
	}

	// Act
	err := repo.Append("wf-1", entries...)
	require.NoError(t, err)

	loaded, err := repo.LoadAll("wf-1")
	require.NoError(t, err)

	// Assert
	require.Len(t, loaded, 3)
	assert.Equal(t, uint64(1), loaded[0].Sequence)
	assert.Equal(t, uint64(2), loaded[1].Sequence)
	assert.Equal(t, uint64(3), loaded[2].Sequence)
}

func TestMemoryJournalRepository_LoadAll_Empty(t *testing.T) {
	// Arrange
	repo := repositories.NewMemoryJournalRepository()

	// Act
	loaded, err := repo.LoadAll("nonexistent")

	// Assert
	require.NoError(t, err)
	assert.Empty(t, loaded)
}

func TestMemoryJournalRepository_LoadAll_SortsBySequence(t *testing.T) {
	// Arrange
	repo := repositories.NewMemoryJournalRepository()
	// Append out of order
	require.NoError(t, repo.Append("wf-1", workflow.JournalEntry{Sequence: 3, Type: workflow.JournalStepCompleted}))
	require.NoError(t, repo.Append("wf-1", workflow.JournalEntry{Sequence: 1, Type: workflow.JournalThreadCreated}))
	require.NoError(t, repo.Append("wf-1", workflow.JournalEntry{Sequence: 2, Type: workflow.JournalStepStarted}))

	// Act
	loaded, err := repo.LoadAll("wf-1")
	require.NoError(t, err)

	// Assert
	require.Len(t, loaded, 3)
	assert.Equal(t, uint64(1), loaded[0].Sequence)
	assert.Equal(t, uint64(2), loaded[1].Sequence)
	assert.Equal(t, uint64(3), loaded[2].Sequence)
}

func TestMemoryJournalRepository_LastSequence(t *testing.T) {
	// Arrange
	repo := repositories.NewMemoryJournalRepository()

	// Assert - empty
	seq, err := repo.LastSequence("wf-1")
	require.NoError(t, err)
	assert.Equal(t, uint64(0), seq)

	// Arrange - add entries
	require.NoError(t, repo.Append("wf-1",
		workflow.JournalEntry{Sequence: 1},
		workflow.JournalEntry{Sequence: 5},
		workflow.JournalEntry{Sequence: 3},
	))

	// Act
	seq, err = repo.LastSequence("wf-1")
	require.NoError(t, err)

	// Assert
	assert.Equal(t, uint64(5), seq)
}

func TestMemoryJournalRepository_IsolatesWorkflows(t *testing.T) {
	// Arrange
	repo := repositories.NewMemoryJournalRepository()
	require.NoError(t, repo.Append("wf-1", workflow.JournalEntry{Sequence: 1, Type: workflow.JournalThreadCreated}))
	require.NoError(t, repo.Append("wf-2", workflow.JournalEntry{Sequence: 1, Type: workflow.JournalStepStarted}))

	// Act
	entries1, err := repo.LoadAll("wf-1")
	require.NoError(t, err)
	entries2, err := repo.LoadAll("wf-2")
	require.NoError(t, err)

	// Assert
	require.Len(t, entries1, 1)
	assert.Equal(t, workflow.JournalThreadCreated, entries1[0].Type)
	require.Len(t, entries2, 1)
	assert.Equal(t, workflow.JournalStepStarted, entries2[0].Type)
}
