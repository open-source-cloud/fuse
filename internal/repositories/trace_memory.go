package repositories

import (
	"sync"

	"github.com/open-source-cloud/fuse/internal/workflow"
)

// MemoryTraceRepository is an in-memory implementation of TraceRepository
type MemoryTraceRepository struct {
	mu      sync.RWMutex
	traces  map[string]*workflow.ExecutionTrace // workflowID -> trace
	schemas map[string][]string                 // schemaID -> []workflowID (insertion order)
}

// NewMemoryTraceRepository creates a new in-memory trace repository
func NewMemoryTraceRepository() *MemoryTraceRepository {
	return &MemoryTraceRepository{
		traces:  make(map[string]*workflow.ExecutionTrace),
		schemas: make(map[string][]string),
	}
}

// Save persists or updates a workflow execution trace
func (r *MemoryTraceRepository) Save(trace *workflow.ExecutionTrace) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, existed := r.traces[trace.WorkflowID]
	r.traces[trace.WorkflowID] = trace

	if !existed {
		r.schemas[trace.SchemaID] = append(r.schemas[trace.SchemaID], trace.WorkflowID)
	}
	return nil
}

// FindByWorkflowID retrieves the trace for a specific workflow execution
func (r *MemoryTraceRepository) FindByWorkflowID(workflowID string) (*workflow.ExecutionTrace, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	trace, exists := r.traces[workflowID]
	if !exists {
		return nil, ErrTraceNotFound
	}
	return trace, nil
}

// FindBySchemaID retrieves traces for all executions of a schema with filtering and pagination
func (r *MemoryTraceRepository) FindBySchemaID(schemaID string, opts TraceQueryOpts) (*TraceQueryResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	wfIDs := r.schemas[schemaID]
	if len(wfIDs) == 0 {
		return &TraceQueryResult{Traces: []*workflow.ExecutionTrace{}, Total: 0}, nil
	}

	// Apply filters
	filtered := make([]*workflow.ExecutionTrace, 0, len(wfIDs))
	for _, wfID := range wfIDs {
		trace := r.traces[wfID]
		if trace == nil {
			continue
		}
		if opts.Status != nil && trace.Status.String() != *opts.Status {
			continue
		}
		if opts.Since != nil && trace.TriggeredAt.Before(*opts.Since) {
			continue
		}
		filtered = append(filtered, trace)
	}

	total := len(filtered)

	// Apply pagination
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := max(opts.Offset, 0)
	if offset >= len(filtered) {
		return &TraceQueryResult{Traces: []*workflow.ExecutionTrace{}, Total: total}, nil
	}
	end := min(offset+limit, len(filtered))

	return &TraceQueryResult{
		Traces: filtered[offset:end],
		Total:  total,
	}, nil
}

// Delete removes a trace
func (r *MemoryTraceRepository) Delete(workflowID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	trace, exists := r.traces[workflowID]
	if !exists {
		return nil
	}

	delete(r.traces, workflowID)

	// Remove from schema index
	wfIDs := r.schemas[trace.SchemaID]
	for i, id := range wfIDs {
		if id == workflowID {
			r.schemas[trace.SchemaID] = append(wfIDs[:i], wfIDs[i+1:]...)
			break
		}
	}

	return nil
}
