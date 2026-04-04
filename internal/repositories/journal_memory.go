package repositories

import (
	"sort"
	"sync"

	"github.com/open-source-cloud/fuse/internal/workflow"
)

// MemoryJournalRepository is an in-memory implementation of JournalRepository
type MemoryJournalRepository struct {
	mu       sync.RWMutex
	journals map[string][]workflow.JournalEntry
}

// NewMemoryJournalRepository creates a new in-memory JournalRepository
func NewMemoryJournalRepository() JournalRepository {
	return &MemoryJournalRepository{
		journals: make(map[string][]workflow.JournalEntry),
	}
}

// Append persists one or more journal entries for a workflow
func (m *MemoryJournalRepository) Append(workflowID string, entries ...workflow.JournalEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.journals[workflowID] = append(m.journals[workflowID], entries...)
	return nil
}

// LoadAll retrieves the full journal for a workflow, ordered by sequence
func (m *MemoryJournalRepository) LoadAll(workflowID string) ([]workflow.JournalEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entries := m.journals[workflowID]
	if entries == nil {
		return []workflow.JournalEntry{}, nil
	}
	cp := make([]workflow.JournalEntry, len(entries))
	copy(cp, entries)
	sort.Slice(cp, func(i, j int) bool {
		return cp[i].Sequence < cp[j].Sequence
	})
	return cp, nil
}

// FindFailed returns all step:failed journal entries for the given workflow.
func (m *MemoryJournalRepository) FindFailed(workflowID string) ([]workflow.JournalEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var failed []workflow.JournalEntry
	for _, e := range m.journals[workflowID] {
		if e.Type == workflow.JournalStepFailed {
			failed = append(failed, e)
		}
	}
	return failed, nil
}

// LastSequence returns the highest sequence number for a workflow
func (m *MemoryJournalRepository) LastSequence(workflowID string) (uint64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entries := m.journals[workflowID]
	if len(entries) == 0 {
		return 0, nil
	}
	var maxSeq uint64
	for _, e := range entries {
		if e.Sequence > maxSeq {
			maxSeq = e.Sequence
		}
	}
	return maxSeq, nil
}
