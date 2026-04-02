package messaging

import (
	"fmt"

	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// CancelWorkflowMessage defines a CancelWorkflow message
type CancelWorkflowMessage struct {
	WorkflowID workflow.ID
	Reason     string
}

// NewCancelWorkflowMessage creates a new CancelWorkflow message
func NewCancelWorkflowMessage(workflowID workflow.ID, reason string) Message {
	return Message{
		Type: CancelWorkflow,
		Args: CancelWorkflowMessage{
			WorkflowID: workflowID,
			Reason:     reason,
		},
	}
}

// CancelWorkflowMessage helper func to cast from a generic Message type
func (m Message) CancelWorkflowMessage() (CancelWorkflowMessage, error) {
	if m.Type != CancelWorkflow {
		return CancelWorkflowMessage{}, fmt.Errorf("message type %s is not CancelWorkflow", m.Type)
	}
	return m.Args.(CancelWorkflowMessage), nil
}
