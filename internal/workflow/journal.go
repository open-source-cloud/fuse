package workflow

import (
	"sync"
	"time"

	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// JournalEntryType classifies what happened at this step
type JournalEntryType string

const (
	// JournalStepStarted a function step has started execution
	JournalStepStarted JournalEntryType = "step:started"
	// JournalStepCompleted a function step completed successfully
	JournalStepCompleted JournalEntryType = "step:completed"
	// JournalStepFailed a function step failed
	JournalStepFailed JournalEntryType = "step:failed"
	// JournalStepRetrying a function step is being retried
	JournalStepRetrying JournalEntryType = "step:retrying"
	// JournalThreadCreated a new thread was created
	JournalThreadCreated JournalEntryType = "thread:created"
	// JournalThreadDone a thread has finished
	JournalThreadDone JournalEntryType = "thread:finished"
	// JournalStateChanged the workflow state changed
	JournalStateChanged JournalEntryType = "state:changed"
	// JournalSleepStarted a sleep action has started
	JournalSleepStarted JournalEntryType = "sleep:started"
	// JournalSleepCompleted a sleep action completed
	JournalSleepCompleted JournalEntryType = "sleep:completed"
	// JournalAwakeableCreated an awakeable was created (waiting for external event)
	JournalAwakeableCreated JournalEntryType = "awakeable:created"
	// JournalAwakeableResolved an awakeable was resolved by an external system
	JournalAwakeableResolved JournalEntryType = "awakeable:resolved"
	// JournalSubWorkflowStarted a sub-workflow was started
	JournalSubWorkflowStarted JournalEntryType = "subworkflow:started"
	// JournalSubWorkflowCompleted a sub-workflow completed
	JournalSubWorkflowCompleted JournalEntryType = "subworkflow:completed"
)

// JournalEntry is a single recorded event in the execution journal
type JournalEntry struct {
	Sequence       uint64                   `json:"sequence"`
	Timestamp      time.Time                `json:"timestamp"`
	Type           JournalEntryType         `json:"type"`
	ThreadID       uint16                   `json:"threadId"`
	FunctionNodeID string                   `json:"functionNodeId,omitempty"`
	ExecID         string                   `json:"execId,omitempty"`
	Input          map[string]any           `json:"input,omitempty"`
	Result         *workflow.FunctionResult `json:"result,omitempty"`
	State          State                    `json:"state,omitempty"`
	ParentThreads  []uint16                 `json:"parentThreads,omitempty"`
	Data           map[string]any           `json:"data,omitempty"`
}

// Journal is an append-only execution log that enables replay
type Journal struct {
	mu            sync.Mutex
	entries       []JournalEntry
	seq           uint64
	lastPersisted uint64
}

// NewJournal creates a new empty Journal
func NewJournal() *Journal {
	return &Journal{entries: make([]JournalEntry, 0, 32)}
}

// Append adds a new entry to the journal, auto-assigning sequence and timestamp
func (j *Journal) Append(entry JournalEntry) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.seq++
	entry.Sequence = j.seq
	entry.Timestamp = time.Now()
	j.entries = append(j.entries, entry)
}

// Entries returns a copy of all journal entries
func (j *Journal) Entries() []JournalEntry {
	j.mu.Lock()
	defer j.mu.Unlock()
	cp := make([]JournalEntry, len(j.entries))
	copy(cp, j.entries)
	return cp
}

// LoadFrom replaces journal contents with persisted entries (for replay)
func (j *Journal) LoadFrom(entries []JournalEntry) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.entries = entries
	if len(entries) > 0 {
		j.seq = entries[len(entries)-1].Sequence
	}
	j.lastPersisted = j.seq
}

// LastSequence returns the highest sequence number in the journal
func (j *Journal) LastSequence() uint64 {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.seq
}

// NewEntries returns journal entries added since the last call to MarkPersisted.
// Used by the handler to determine what needs to be persisted.
func (j *Journal) NewEntries() []JournalEntry {
	j.mu.Lock()
	defer j.mu.Unlock()
	var newEntries []JournalEntry
	for _, e := range j.entries {
		if e.Sequence > j.lastPersisted {
			newEntries = append(newEntries, e)
		}
	}
	return newEntries
}

// MarkPersisted marks all current entries as persisted
func (j *Journal) MarkPersisted() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.lastPersisted = j.seq
}
