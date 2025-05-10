package messaging

import "fmt"

type MessageType string

const (
	ActorInit MessageType = "actor:init"
	ChildInit MessageType = "actor:child:init"
	TriggerWorkflow MessageType = "workflow:trigger"
)

type (
	Message struct {
		Type   MessageType
		Data   any
	}

	TriggerWorkflowMessage struct {
		SchemaID string
	}
)

func NewActorInitMessage(data any) Message {
	return Message{
		Type:   ActorInit,
		Data:   data,
	}
}

func NewChildInitMessage(data any) Message {
	return Message{
		Type:   ChildInit,
		Data:   data,
	}
}

func NewTriggerWorkflowMessage(schemaID string) Message {
	return Message{
		Type:   TriggerWorkflow,
		Data: TriggerWorkflowMessage{
			SchemaID: schemaID,
		},
	}
}

func (m Message) TriggerWorkflowMessage() (TriggerWorkflowMessage, error) {
	if m.Type != TriggerWorkflow {
		return TriggerWorkflowMessage{}, fmt.Errorf("message type %s is not TriggerWorkflow", m.Type)
	}
	return m.Data.(TriggerWorkflowMessage), nil
}
