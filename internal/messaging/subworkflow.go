package messaging

import "github.com/open-source-cloud/fuse/pkg/workflow"

// SubWorkflowCompletedMessage defines a SubWorkflowCompleted message
type SubWorkflowCompletedMessage struct {
	ParentWorkflowID workflow.ID
	ParentThreadID   uint16
	ParentExecID     workflow.ExecID
	ChildWorkflowID  workflow.ID
	ChildFinalState  string
	ChildOutput      map[string]any
}

// NewSubWorkflowCompletedMessage creates a new SubWorkflowCompleted message
func NewSubWorkflowCompletedMessage(
	parentWorkflowID workflow.ID,
	parentThreadID uint16,
	parentExecID workflow.ExecID,
	childWorkflowID workflow.ID,
	childFinalState string,
	childOutput map[string]any,
) Message {
	return Message{
		Type: SubWorkflowCompleted,
		Args: SubWorkflowCompletedMessage{
			ParentWorkflowID: parentWorkflowID,
			ParentThreadID:   parentThreadID,
			ParentExecID:     parentExecID,
			ChildWorkflowID:  childWorkflowID,
			ChildFinalState:  childFinalState,
			ChildOutput:      childOutput,
		},
	}
}
