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
		Type MessageType
		Args any
	}

	TriggerWorkflowMessage struct {
		SchemaID string
	}
)

func NewActorInitMessage(args any) Message {
	return Message{
		Type: ActorInit,
		Args: args,
	}
}

func NewChildInitMessage(args any) Message {
	return Message{
		Type: ChildInit,
		Args: args,
	}
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
