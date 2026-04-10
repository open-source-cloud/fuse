package repositories

import (
	"time"

	"github.com/open-source-cloud/fuse/internal/workflow"
)

// ExecutionListItem is a lightweight projection of a workflow for list endpoints.
type ExecutionListItem struct {
	WorkflowID string    `json:"workflowId"`
	SchemaID   string    `json:"schemaId"`
	State      string    `json:"state"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// ExecutionListFilter defines the query parameters for listing executions.
type ExecutionListFilter struct {
	SchemaID string
	Status   string    // optional filter by state
	From     time.Time // optional: created_at >= from
	To       time.Time // optional: created_at <= to
	Page     int
	Size     int
}

// ExecutionListResult is a paginated result for listing executions.
type ExecutionListResult struct {
	Items    []ExecutionListItem `json:"items"`
	Total    int                 `json:"total"`
	Page     int                 `json:"page"`
	Size     int                 `json:"size"`
	LastPage int                 `json:"lastPage"`
}

type (
	// WorkflowRepository defines the interface o a WorkflowRepository repository
	WorkflowRepository interface {
		Exists(id string) bool
		Get(id string) (*workflow.Workflow, error)
		Save(workflow *workflow.Workflow) error
		// FindByState returns workflow IDs for workflows in any of the given states
		FindByState(states ...workflow.State) ([]string, error)
		// SaveSubWorkflowRef stores a parent-child workflow relationship
		SaveSubWorkflowRef(ref *workflow.SubWorkflowRef) error
		// FindSubWorkflowRef finds a sub-workflow reference by child workflow ID
		FindSubWorkflowRef(childID string) (*workflow.SubWorkflowRef, error)
		// FindActiveSubWorkflows finds all sub-workflow references for a parent
		FindActiveSubWorkflows(parentID string) ([]*workflow.SubWorkflowRef, error)
		// GetSnapshotRef returns the object store key of the execution snapshot (empty if not set)
		GetSnapshotRef(workflowID string) (string, error)
		// SetSnapshotRef records the object store key of the execution snapshot
		SetSnapshotRef(workflowID string, snapshotRef string) error
		// FindExecutions returns a paginated list of workflow executions filtered by schema, status, and time range.
		FindExecutions(filter ExecutionListFilter) (*ExecutionListResult, error)
	}
)
