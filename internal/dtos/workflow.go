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
