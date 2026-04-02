package messaging

import "github.com/open-source-cloud/fuse/pkg/workflow"

// AwakeableResolvedMessage defines an AwakeableResolved message
type AwakeableResolvedMessage struct {
	WorkflowID  workflow.ID
	AwakeableID string
	ExecID      workflow.ExecID
	ThreadID    uint16
	Data        map[string]any
}

// NewAwakeableResolvedMessage creates a new AwakeableResolved message
func NewAwakeableResolvedMessage(
	workflowID workflow.ID,
	awakeableID string,
	execID workflow.ExecID,
	threadID uint16,
	data map[string]any,
) Message {
	return Message{
		Type: AwakeableResolvedMsg,
		Args: AwakeableResolvedMessage{
			WorkflowID:  workflowID,
			AwakeableID: awakeableID,
			ExecID:      execID,
			ThreadID:    threadID,
			Data:        data,
		},
	}
}
