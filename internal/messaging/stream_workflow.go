package messaging

import (
	"fmt"

	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// StreamWorkflowMessage defines a message to register/unregister streaming callbacks
type StreamWorkflowMessage struct {
	WorkflowID workflow.ID
	Callback   workflow.StreamCallback // nil to unregister
}

// NewStreamWorkflowMessage creates a new StreamWorkflow message
func NewStreamWorkflowMessage(workflowID workflow.ID, callback workflow.StreamCallback) Message {
	return Message{
		Type: StreamWorkflow,
		Args: StreamWorkflowMessage{
			WorkflowID: workflowID,
			Callback:   callback,
		},
	}
}

// StreamWorkflowMessage helper func to cast from a generic Message type
func (m Message) StreamWorkflowMessage() (StreamWorkflowMessage, error) {
	if m.Type != StreamWorkflow {
		return StreamWorkflowMessage{}, fmt.Errorf("message type %s is not StreamWorkflow", m.Type)
	}
	return m.Args.(StreamWorkflowMessage), nil
}
