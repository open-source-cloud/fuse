package repositories

import (
	"fmt"
	"sync"

	"github.com/open-source-cloud/fuse/internal/workflow"
)

// NewMemoryWorkflowRepository creates a new in-memory WorkflowRepository repository
func NewMemoryWorkflowRepository() WorkflowRepository {
	return &DefaultWorkflowRepository{
		workflows: make(map[string]*workflow.Workflow),
	}
}

type (
	// WorkflowRepository defines the interface o a WorkflowRepository repository
	WorkflowRepository interface {
		Exists(id string) bool
		Get(id string) (*workflow.Workflow, error)
		Save(workflow *workflow.Workflow) error
	}

	// DefaultWorkflowRepository is the default implementation of the WorkflowRepository interface (in-memory)
	DefaultWorkflowRepository struct {
		mu        sync.RWMutex
		workflows map[string]*workflow.Workflow
	}
)

// Exists checks if a workflow exists in the repository
func (m *DefaultWorkflowRepository) Exists(id string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.workflows[id]
	return ok
}

// Get retrieves a workflow from the repository
func (m *DefaultWorkflowRepository) Get(id string) (*workflow.Workflow, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	workflow, ok := m.workflows[id]
	if !ok {
		return nil, fmt.Errorf("workflow %s not found", id)
	}
	return workflow, nil
}

// Save stores a workflow in the repository
func (m *DefaultWorkflowRepository) Save(workflow *workflow.Workflow) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.workflows[workflow.ID().String()] = workflow
	return nil
}
