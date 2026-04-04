package repositories

import (
	"github.com/open-source-cloud/fuse/internal/workflow"
)

// JournalRepository defines the interface for persisting workflow execution journals
type JournalRepository interface {
	// Append persists one or more journal entries for a workflow
	Append(workflowID string, entries ...workflow.JournalEntry) error

	// LoadAll retrieves the full journal for a workflow, ordered by sequence
	LoadAll(workflowID string) ([]workflow.JournalEntry, error)

	// LastSequence returns the highest sequence number for a workflow
	LastSequence(workflowID string) (uint64, error)

	// FindFailed returns all step:failed journal entries for the given workflow
	FindFailed(workflowID string) ([]workflow.JournalEntry, error)
}
