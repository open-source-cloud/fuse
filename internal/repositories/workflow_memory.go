package repositories

import (
	"fmt"
	"sync"

	"github.com/open-source-cloud/fuse/internal/workflow"
)

// MemoryWorkflowRepository is the default implementation of the WorkflowRepository interface (in-memory)
type MemoryWorkflowRepository struct {
	WorkflowRepository
	mu              sync.RWMutex
	workflows       map[string]*workflow.Workflow
	subWorkflowRefs map[string]*workflow.SubWorkflowRef // childID -> ref
	parentChildren  map[string][]string                 // parentID -> []childID
	snapshotRefs    map[string]string                   // workflowID -> snapshot ref
}

// NewMemoryWorkflowRepository creates a new in-memory WorkflowRepository repository
func NewMemoryWorkflowRepository() WorkflowRepository {
	return &MemoryWorkflowRepository{
		workflows:       make(map[string]*workflow.Workflow),
		subWorkflowRefs: make(map[string]*workflow.SubWorkflowRef),
		parentChildren:  make(map[string][]string),
		snapshotRefs:    make(map[string]string),
	}
}

// Exists checks if a workflow exists in the repository
func (m *MemoryWorkflowRepository) Exists(id string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.workflows[id]
	return ok
}

// Get retrieves a workflow from the repository
func (m *MemoryWorkflowRepository) Get(id string) (*workflow.Workflow, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	workflow, ok := m.workflows[id]
	if !ok {
		return nil, fmt.Errorf("workflow %s not found", id)
	}
	return workflow, nil
}

// Save stores a workflow in the repository
func (m *MemoryWorkflowRepository) Save(workflow *workflow.Workflow) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.workflows[workflow.ID().String()] = workflow
	return nil
}

// FindByState returns workflow IDs for workflows in any of the given states
func (m *MemoryWorkflowRepository) FindByState(states ...workflow.State) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var ids []string
	for id, wf := range m.workflows {
		for _, state := range states {
			if wf.State() == state {
				ids = append(ids, id)
				break
			}
		}
	}
	return ids, nil
}

// SaveSubWorkflowRef stores a parent-child workflow relationship
func (m *MemoryWorkflowRepository) SaveSubWorkflowRef(ref *workflow.SubWorkflowRef) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	childID := ref.ChildWorkflowID.String()
	parentID := ref.ParentWorkflowID.String()
	m.subWorkflowRefs[childID] = ref
	m.parentChildren[parentID] = append(m.parentChildren[parentID], childID)
	return nil
}

// FindSubWorkflowRef finds a sub-workflow reference by child workflow ID
func (m *MemoryWorkflowRepository) FindSubWorkflowRef(childID string) (*workflow.SubWorkflowRef, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ref, exists := m.subWorkflowRefs[childID]
	if !exists {
		return nil, fmt.Errorf("sub-workflow ref for child %s not found", childID)
	}
	return ref, nil
}

// GetSnapshotRef returns the object store key of the execution snapshot.
func (m *MemoryWorkflowRepository) GetSnapshotRef(workflowID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.snapshotRefs[workflowID], nil
}

// SetSnapshotRef records the object store key of the execution snapshot.
func (m *MemoryWorkflowRepository) SetSnapshotRef(workflowID string, snapshotRef string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.snapshotRefs[workflowID] = snapshotRef
	return nil
}

// FindActiveSubWorkflows finds all sub-workflow references for a parent
func (m *MemoryWorkflowRepository) FindActiveSubWorkflows(parentID string) ([]*workflow.SubWorkflowRef, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	childIDs := m.parentChildren[parentID]
	var refs []*workflow.SubWorkflowRef
	for _, childID := range childIDs {
		if ref, exists := m.subWorkflowRefs[childID]; exists {
			refs = append(refs, ref)
		}
	}
	return refs, nil
}
