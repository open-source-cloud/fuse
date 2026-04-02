package workflow

import (
	"time"

	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// AwakeableStatus represents the status of an awakeable
type AwakeableStatus string

const (
	// AwakeablePending the awakeable is waiting to be resolved
	AwakeablePending AwakeableStatus = "pending"
	// AwakeableResolved the awakeable has been resolved by an external system
	AwakeableResolved AwakeableStatus = "resolved"
	// AwakeableTimedOut the awakeable timed out before being resolved
	AwakeableTimedOut AwakeableStatus = "timed_out"
	// AwakeableCancelled the awakeable was cancelled (e.g., workflow cancelled)
	AwakeableCancelled AwakeableStatus = "cancelled"
)

// Awakeable represents a durable promise that can be resolved externally
type Awakeable struct {
	ID         string          `json:"id"`
	WorkflowID workflow.ID     `json:"workflowId"`
	ExecID     workflow.ExecID `json:"execId"`
	ThreadID   uint16          `json:"threadId"`
	CreatedAt  time.Time       `json:"createdAt"`
	Timeout    time.Duration   `json:"timeout"`
	DeadlineAt time.Time       `json:"deadlineAt"`
	Status     AwakeableStatus `json:"status"`
	Result     map[string]any  `json:"result,omitempty"`
}
