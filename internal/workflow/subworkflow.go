package workflow

import "github.com/open-source-cloud/fuse/pkg/workflow"

// SubWorkflowRef tracks a parent-child workflow relationship
type SubWorkflowRef struct {
	ParentWorkflowID workflow.ID     `json:"parentWorkflowId"`
	ParentThreadID   uint16          `json:"parentThreadId"`
	ParentExecID     workflow.ExecID `json:"parentExecId"`
	ChildWorkflowID  workflow.ID     `json:"childWorkflowId"`
	ChildSchemaID    string          `json:"childSchemaId"`
	Async            bool            `json:"async"`
}
