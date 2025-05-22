package repos

import (
	"fmt"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/pkg/store"
)

// NewMemoryWorkflowRepo creates a new in-memory WorkflowRepo repository
func NewMemoryWorkflowRepo() WorkflowRepo {
	return &memoryWorkflowRepo{
		workflows: store.New(),
	}
}

type (
	// WorkflowRepo defines the interface o a WorkflowRepo repository
	WorkflowRepo interface {
		Exists(id string) bool
		Get(id string) (*workflow.Workflow, error)
		Save(workflow *workflow.Workflow) error
	}

	memoryWorkflowRepo struct {
		workflows *store.KV
	}
)

func (m *memoryWorkflowRepo) Exists(id string) bool {
	return m.workflows.Has(id)
}

func (m *memoryWorkflowRepo) Get(id string) (*workflow.Workflow, error) {
	foundWorkflow := m.workflows.Get(id)
	if foundWorkflow == nil {
		return nil, fmt.Errorf("workflow %s not found", id)
	}
	return foundWorkflow.(*workflow.Workflow), nil
}

func (m *memoryWorkflowRepo) Save(workflow *workflow.Workflow) error {
	m.workflows.Set(workflow.ID().String(), workflow)
	return nil
}
