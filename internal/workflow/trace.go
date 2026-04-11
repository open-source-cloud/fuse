package workflow

import (
	"time"

	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// ExecutionTrace is the complete, persistable trace of a workflow execution
type ExecutionTrace struct {
	WorkflowID  string               `json:"workflowId"`
	SchemaID    string               `json:"schemaId"`
	Status      State                `json:"status"`
	TriggeredAt time.Time            `json:"triggeredAt"`
	CompletedAt *time.Time           `json:"completedAt,omitempty"`
	Duration    *string              `json:"duration,omitempty" example:"5s"`
	Steps       []ExecutionStepTrace `json:"steps"`
	Error       *string              `json:"error,omitempty"`
}

// ExecutionStepTrace is the trace for a single step (node execution)
type ExecutionStepTrace struct {
	ExecID         string                   `json:"execId"`
	ThreadID       uint16                   `json:"threadId"`
	FunctionNodeID string                   `json:"functionNodeId"`
	StartedAt      time.Time                `json:"startedAt"`
	CompletedAt    *time.Time               `json:"completedAt,omitempty"`
	Duration       *string                  `json:"duration,omitempty" example:"2s"`
	Input          map[string]any           `json:"input,omitempty"`
	Output         *workflow.FunctionOutput `json:"output,omitempty"`
	Status         string                   `json:"status"`
	Attempt        int                      `json:"attempt"`
	Error          *string                  `json:"error,omitempty"`
}

// TraceRetentionConfig defines retention policy for execution traces
type TraceRetentionConfig struct {
	MaxAge   time.Duration `json:"maxAge,omitempty"`
	MaxCount int           `json:"maxCount,omitempty"`
}
