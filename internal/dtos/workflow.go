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
