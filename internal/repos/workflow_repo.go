package repos

import "github.com/open-source-cloud/fuse/internal/workflow"

func NewMemoryWorkflowRepo() WorkflowRepo {
	return &memoryWorkflowRepo{
		workflows: make(map[string]workflow.Workflow),
	}
}

type (
	WorkflowRepo interface {
		Exists(id string) bool
		Get(id string) (*workflow.Workflow, error)
		Save(workflow *workflow.Workflow) error
	}

	memoryWorkflowRepo struct {
		workflows map[string]workflow.Workflow
	}
)

func (m *memoryWorkflowRepo) Exists(id string) bool {
	_, ok := m.workflows[id]
	return ok
}

func (m *memoryWorkflowRepo) Get(id string) (*workflow.Workflow, error) {
	foundWorkflow, ok := m.workflows[id]
	if !ok {
		return nil, nil
	}
	return &foundWorkflow, nil
}

func (m *memoryWorkflowRepo) Save(workflow *workflow.Workflow) error {
	m.workflows[workflow.ID.String()] = *workflow
	return nil
}
