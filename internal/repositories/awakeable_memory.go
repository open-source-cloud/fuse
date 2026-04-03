package repositories

import (
	"sync"

	workflow "github.com/open-source-cloud/fuse/internal/workflow"
)

// MemoryAwakeableRepository is an in-memory implementation of AwakeableRepository
type MemoryAwakeableRepository struct {
	mu         sync.RWMutex
	awakeables map[string]*workflow.Awakeable
}

// NewMemoryAwakeableRepository creates a new MemoryAwakeableRepository
func NewMemoryAwakeableRepository() *MemoryAwakeableRepository {
	return &MemoryAwakeableRepository{
		awakeables: make(map[string]*workflow.Awakeable),
	}
}

// Save stores an awakeable
func (r *MemoryAwakeableRepository) Save(awakeable *workflow.Awakeable) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.awakeables[awakeable.ID] = awakeable
	return nil
}

// FindByID retrieves an awakeable by its ID
func (r *MemoryAwakeableRepository) FindByID(id string) (*workflow.Awakeable, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	awakeable, exists := r.awakeables[id]
	if !exists {
		return nil, ErrAwakeableNotFound
	}
	return awakeable, nil
}

// FindPending retrieves all pending awakeables for a given workflow
func (r *MemoryAwakeableRepository) FindPending(workflowID string) ([]*workflow.Awakeable, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*workflow.Awakeable
	for _, a := range r.awakeables {
		if a.WorkflowID.String() == workflowID && a.Status == workflow.AwakeablePending {
			result = append(result, a)
		}
	}
	return result, nil
}

// Resolve resolves an awakeable with the given result data
func (r *MemoryAwakeableRepository) Resolve(id string, result map[string]any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	awakeable, exists := r.awakeables[id]
	if !exists {
		return ErrAwakeableNotFound
	}
	awakeable.Status = workflow.AwakeableResolved
	awakeable.Result = result
	return nil
}
