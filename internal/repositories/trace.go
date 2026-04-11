package repositories

import (
	"errors"
	"time"

	"github.com/open-source-cloud/fuse/internal/workflow"
)

var (
	// ErrTraceNotFound is returned when a trace is not found
	ErrTraceNotFound = errors.New("trace not found")
)

// TraceQueryOpts defines query options for listing traces
type TraceQueryOpts struct {
	Limit  int
	Offset int
	Status *string
	Since  *time.Time
}

// TraceQueryResult wraps trace query results with pagination info
type TraceQueryResult struct {
	Traces []*workflow.ExecutionTrace
	Total  int
}

// TraceRepository defines the interface for persisting and querying execution traces
type TraceRepository interface {
	// Save persists or updates a workflow execution trace
	Save(trace *workflow.ExecutionTrace) error
	// FindByWorkflowID retrieves the trace for a specific workflow execution
	FindByWorkflowID(workflowID string) (*workflow.ExecutionTrace, error)
	// FindBySchemaID retrieves traces for all executions of a schema
	FindBySchemaID(schemaID string, opts TraceQueryOpts) (*TraceQueryResult, error)
	// Delete removes a trace
	Delete(workflowID string) error
}
