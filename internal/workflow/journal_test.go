package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJournal_Append(t *testing.T) {
	// Arrange
	j := NewJournal()

	// Act
	j.Append(JournalEntry{Type: JournalThreadCreated, ThreadID: 0})
	j.Append(JournalEntry{Type: JournalStepStarted, ThreadID: 0, FunctionNodeID: "node-1"})
	j.Append(JournalEntry{Type: JournalStepCompleted, ThreadID: 0, FunctionNodeID: "node-1"})

	// Assert
	entries := j.Entries()
	require.Len(t, entries, 3)
	assert.Equal(t, uint64(1), entries[0].Sequence)
	assert.Equal(t, uint64(2), entries[1].Sequence)
	assert.Equal(t, uint64(3), entries[2].Sequence)
	assert.Equal(t, JournalThreadCreated, entries[0].Type)
	assert.Equal(t, JournalStepStarted, entries[1].Type)
	assert.Equal(t, JournalStepCompleted, entries[2].Type)
	assert.False(t, entries[0].Timestamp.IsZero())
}

func TestJournal_Entries_ReturnsCopy(t *testing.T) {
	// Arrange
	j := NewJournal()
	j.Append(JournalEntry{Type: JournalThreadCreated})

	// Act
	entries := j.Entries()
	entries[0].Type = JournalStepFailed // mutate the copy

	// Assert - original should be unchanged
	original := j.Entries()
	assert.Equal(t, JournalThreadCreated, original[0].Type)
}

func TestJournal_LoadFrom(t *testing.T) {
	// Arrange
	j := NewJournal()
	persisted := []JournalEntry{
		{Sequence: 1, Type: JournalThreadCreated},
		{Sequence: 2, Type: JournalStepStarted},
		{Sequence: 3, Type: JournalStepCompleted},
	}

	// Act
	j.LoadFrom(persisted)

	// Assert
	entries := j.Entries()
	require.Len(t, entries, 3)
	assert.Equal(t, uint64(3), j.LastSequence())
}

func TestJournal_LoadFrom_Empty(t *testing.T) {
	// Arrange
	j := NewJournal()

	// Act
	j.LoadFrom([]JournalEntry{})

	// Assert
	assert.Equal(t, uint64(0), j.LastSequence())
	assert.Empty(t, j.Entries())
}

func TestJournal_NewEntries(t *testing.T) {
	// Arrange
	j := NewJournal()
	j.Append(JournalEntry{Type: JournalThreadCreated})
	j.Append(JournalEntry{Type: JournalStepStarted})
	j.MarkPersisted()

	// Act - add more entries after persisting
	j.Append(JournalEntry{Type: JournalStepCompleted})
	newEntries := j.NewEntries()

	// Assert
	require.Len(t, newEntries, 1)
	assert.Equal(t, JournalStepCompleted, newEntries[0].Type)
	assert.Equal(t, uint64(3), newEntries[0].Sequence)
}

func TestJournal_NewEntries_AfterLoadFrom(t *testing.T) {
	// Arrange - simulate loading from persistence then adding new entries
	j := NewJournal()
	j.LoadFrom([]JournalEntry{
		{Sequence: 1, Type: JournalThreadCreated},
		{Sequence: 2, Type: JournalStepStarted},
	})

	// Act - new entries after load should be tracked
	j.Append(JournalEntry{Type: JournalStepCompleted})
	newEntries := j.NewEntries()

	// Assert
	require.Len(t, newEntries, 1)
	assert.Equal(t, JournalStepCompleted, newEntries[0].Type)
}

func TestJournal_LastSequence(t *testing.T) {
	// Arrange
	j := NewJournal()

	// Assert - empty
	assert.Equal(t, uint64(0), j.LastSequence())

	// Act
	j.Append(JournalEntry{Type: JournalThreadCreated})
	j.Append(JournalEntry{Type: JournalStepStarted})

	// Assert
	assert.Equal(t, uint64(2), j.LastSequence())
}
