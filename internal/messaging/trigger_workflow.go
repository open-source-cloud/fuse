package messaging

import "fmt"

// TriggerWorkflowMessage defines a TriggerWorkflow message
type TriggerWorkflowMessage struct {
	SchemaID string
}

// NewTriggerWorkflowMessage creates a new TriggerWorkflow message
func NewTriggerWorkflowMessage(schemaID string) Message {
	return Message{
		Type:   TriggerWorkflow,
		Args: TriggerWorkflowMessage{
			SchemaID: schemaID,
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
