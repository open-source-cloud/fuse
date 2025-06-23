package dtos

import "github.com/open-source-cloud/fuse/internal/workflow"

// UpsertWorkflowSchema for updating or creating a workflow.GraphSchema
type UpsertWorkflowSchema struct {
	Nodes    []workflow.NodeSchema `json:"nodes" validate:"required,dive"`
	Edges    []workflow.EdgeSchema `json:"edges" validate:"required,dive"`
	Tags     map[string]string     `json:"tags,omitempty"`
	Metadata map[string]string     `json:"metadata,omitempty"`
}
