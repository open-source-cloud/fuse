package messaging

import "fmt"

type TriggerWorkflowMessage struct {
	SchemaID string
}

func NewTriggerWorkflowMessage(schemaID string) Message {
	return Message{
		Type:   TriggerWorkflow,
		Args: TriggerWorkflowMessage{
			SchemaID: schemaID,
		},
	}
}

func (m Message) TriggerWorkflowMessage() (TriggerWorkflowMessage, error) {
	if m.Type != TriggerWorkflow {
		return TriggerWorkflowMessage{}, fmt.Errorf("message type %s is not TriggerWorkflow", m.Type)
	}
	return m.Args.(TriggerWorkflowMessage), nil
}
