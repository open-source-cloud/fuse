package messaging

import "fmt"

type MessageType string

const (
	ActorInit MessageType = "actor:init"
	ChildInit MessageType = "actor:child:init"
	WorkflowExecuteJSON MessageType = "workflow:executeJson"
)

type (
	Message struct {
		Type   MessageType
		Data   any
	}

	WorkflowExecuteJSONMessage struct {
		JsonBytes []byte
	}
)

func NewActorInitMessage() Message {
	return Message{
		Type:   ActorInit,
	}
}

func NewChildInitMessage(data any) Message {
	return Message{
		Type:   ChildInit,
		Data:   data,
	}
}

func NewWorkflowExecuteJSONMessage(jsonBytes []byte) Message {
	return Message{
		Type:   WorkflowExecuteJSON,
		Data: WorkflowExecuteJSONMessage{
			JsonBytes: jsonBytes,
		},
	}
}

func (m Message) WorkflowExecuteJSONMessage() (WorkflowExecuteJSONMessage, error) {
	if m.Type != WorkflowExecuteJSON {
		return WorkflowExecuteJSONMessage{}, fmt.Errorf("message type %s is not WorkflowExecuteJSON", m.Type)
	}
	return m.Data.(WorkflowExecuteJSONMessage), nil
}
