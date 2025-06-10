package messaging

import (
	"fmt"
	"github.com/open-source-cloud/fuse/internal/workflow"
)

// TriggerWorkflowMessage defines a TriggerWorkflow message
type TriggerWorkflowMessage struct {
	SchemaID   string
	WorkflowID workflow.ID
}

// NewTriggerWorkflowMessage creates a new TriggerWorkflow message
func NewTriggerWorkflowMessage(schemaID string, workflowID workflow.ID) Message {
	return Message{
		Type: TriggerWorkflow,
		Args: TriggerWorkflowMessage{
			SchemaID:   schemaID,
			WorkflowID: workflowID,
		},
	}
}

// TriggerWorkflowMessage helper func to cast from a generic Message type
func (m Message) TriggerWorkflowMessage() (TriggerWorkflowMessage, error) {
	if m.Type != TriggerWorkflow {
		return TriggerWorkflowMessage{}, fmt.Errorf("message type %s is not TriggerWorkflow", m.Type)
	}
	return m.Args.(TriggerWorkflowMessage), nil
}
