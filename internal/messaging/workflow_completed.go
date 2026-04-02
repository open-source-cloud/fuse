package messaging

import (
	"fmt"

	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// WorkflowCompletedMessage defines a WorkflowCompleted message
type WorkflowCompletedMessage struct {
	WorkflowID workflow.ID
	FinalState string
}

// NewWorkflowCompletedMessage creates a new WorkflowCompleted message
func NewWorkflowCompletedMessage(workflowID workflow.ID, finalState string) Message {
	return Message{
		Type: WorkflowCompleted,
		Args: WorkflowCompletedMessage{
			WorkflowID: workflowID,
			FinalState: finalState,
		},
	}
}

// WorkflowCompletedMessage helper func to cast from a generic Message type
func (m Message) WorkflowCompletedMessage() (WorkflowCompletedMessage, error) {
	if m.Type != WorkflowCompleted {
		return WorkflowCompletedMessage{}, fmt.Errorf("message type %s is not WorkflowCompleted", m.Type)
	}
	return m.Args.(WorkflowCompletedMessage), nil
}
