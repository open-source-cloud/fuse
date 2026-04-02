package repositories

import (
	"github.com/open-source-cloud/fuse/internal/workflow"
)

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
	}
)
