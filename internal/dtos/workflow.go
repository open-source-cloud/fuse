package dtos

import "github.com/open-source-cloud/fuse/pkg/workflow"

// UpsertSchemaResponse represents schema upsert response
type UpsertSchemaResponse struct {
	SchemaID string `json:"schemaId" example:"my-workflow-schema"`
}

// TriggerWorkflowRequest represents the data structure for triggering a workflow
type TriggerWorkflowRequest struct {
	SchemaID string `json:"schemaID" validate:"required"`
}

// TriggerWorkflowResponse represents trigger workflow response
type TriggerWorkflowResponse struct {
	SchemaID   string `json:"schemaId" example:"my-schema"`
	WorkflowID string `json:"workflowId" example:"550e8400-e29b-41d4-a716-446655440000"`
	Code       string `json:"code" example:"OK"`
}

// AsyncFunctionRequest is the request body for the AsyncFunctionHandler
type AsyncFunctionRequest struct {
	Result workflow.FunctionOutput `json:"result"`
}

// AsyncFunctionResultResponse represents async function result response
type AsyncFunctionResultResponse struct {
	WorkflowID string `json:"workflowID" example:"550e8400-e29b-41d4-a716-446655440000"`
	ExecID     string `json:"execID" example:"exec-123"`
	Code       string `json:"code" example:"OK"`
}

// CancelWorkflowRequest is the request body for the CancelWorkflowHandler
type CancelWorkflowRequest struct {
	Reason string `json:"reason"`
}

// CancelWorkflowResponse represents cancel workflow response
type CancelWorkflowResponse struct {
	WorkflowID  string `json:"workflowId" example:"550e8400-e29b-41d4-a716-446655440000"`
	Status      string `json:"status" example:"cancelled"`
	CancelledAt string `json:"cancelledAt" example:"2025-07-28T10:00:00Z"`
}

// GetWorkflowResponse represents get workflow status response
type GetWorkflowResponse struct {
	WorkflowID string `json:"workflowId" example:"550e8400-e29b-41d4-a716-446655440000"`
	Status     string `json:"status" example:"finished"`
}

// ResolveAwakeableRequest is the request body for the ResolveAwakeableHandler
type ResolveAwakeableRequest struct {
	Data map[string]any `json:"data"`
}

// ResolveAwakeableResponse represents resolve awakeable response
type ResolveAwakeableResponse struct {
	WorkflowID  string `json:"workflowId" example:"550e8400-e29b-41d4-a716-446655440000"`
	AwakeableID string `json:"awakeableId" example:"awk-123"`
	Status      string `json:"status" example:"resolved"`
}

// RetryNodeRequest is the request body for retrying a specific failed node
type RetryNodeRequest struct {
	ExecID string `json:"execId" validate:"required"`
}

// RetryNodeResponse represents retry node response
type RetryNodeResponse struct {
	WorkflowID string `json:"workflowId" example:"550e8400-e29b-41d4-a716-446655440000"`
	ExecID     string `json:"execId" example:"exec-123"`
	Status     string `json:"status" example:"accepted"`
}

// RetryWorkflowRequest is the request body for retrying an entire workflow
type RetryWorkflowRequest struct {
	// Strategy: "from-scratch" or "from-failed" (default: "from-failed")
	Strategy string `json:"strategy,omitempty" example:"from-failed"`
	// ExecID: optional specific failed exec to retry (only for from-failed strategy)
	ExecID string `json:"execId,omitempty"`
}

// RetryWorkflowResponse represents retry workflow response
type RetryWorkflowResponse struct {
	OriginalWorkflowID string `json:"originalWorkflowId" example:"550e8400-e29b-41d4-a716-446655440000"`
	NewWorkflowID      string `json:"newWorkflowId,omitempty" example:"660e9500-f39c-52e5-b827-557766551111"`
	Status             string `json:"status" example:"accepted"`
}
