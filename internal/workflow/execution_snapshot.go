package workflow

import "time"

// ExecutionSnapshot is a derived, UI-consumable projection of a workflow execution.
// Built from journal entries and aggregated outputs — a single JSON document
// that shows the full execution state.
type ExecutionSnapshot struct {
	SchemaVersion     int                     `json:"schemaVersion"`
	WorkflowID        string                  `json:"workflowId"`
	SchemaID          string                  `json:"schemaId"`
	Status            string                  `json:"status"`
	StartedAt         *time.Time              `json:"startedAt,omitempty"`
	FinishedAt        *time.Time              `json:"finishedAt,omitempty"`
	Error             *string                 `json:"error,omitempty"`
	Threads           []SnapshotThread        `json:"threads"`
	NodeRuns          []SnapshotNodeRun       `json:"nodeRuns"`
	AggregatedOutputs map[string]any          `json:"aggregatedOutputs"`
	Timeline          []SnapshotTimelineEvent `json:"timeline"`
}

// SnapshotThread represents a thread's state in the snapshot.
type SnapshotThread struct {
	ThreadID      uint16   `json:"threadId"`
	ParentThreads []uint16 `json:"parentThreads,omitempty"`
	State         string   `json:"state"`
	LastExecID    string   `json:"lastExecId,omitempty"`
}

// SnapshotNodeRun represents one function execution attempt in the snapshot.
type SnapshotNodeRun struct {
	ExecID               string         `json:"execId"`
	NodeID               string         `json:"nodeId"`
	ThreadID             uint16         `json:"threadId"`
	Input                map[string]any `json:"input,omitempty"`
	Output               map[string]any `json:"output,omitempty"`
	Status               string         `json:"status"`
	StartedAt            *time.Time     `json:"startedAt,omitempty"`
	FinishedAt           *time.Time     `json:"finishedAt,omitempty"`
	RetryAttempt         int            `json:"retryAttempt,omitempty"`
	PreviousExecID       string         `json:"previousExecId,omitempty"`
	JournalSequenceStart uint64         `json:"journalSequenceStart,omitempty"`
	JournalSequenceEnd   uint64         `json:"journalSequenceEnd,omitempty"`
}

// SnapshotTimelineEvent is a slim journal event for the timeline view.
type SnapshotTimelineEvent struct {
	Sequence  uint64    `json:"sequence"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	ThreadID  uint16    `json:"threadId,omitempty"`
	ExecID    string    `json:"execId,omitempty"`
	NodeID    string    `json:"nodeId,omitempty"`
}
