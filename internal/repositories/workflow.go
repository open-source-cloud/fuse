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
	}
)
