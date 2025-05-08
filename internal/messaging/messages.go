package messaging

import "fmt"

type MessageType string

const (
	WorkflowExecuteJSON MessageType = "workflow:executeJson"
)

type (
	Message struct {
		Target string
		Type   MessageType
		Data   any
	}

	WorkflowExecuteJSONMessage struct {
		JsonBytes []byte
	}
)

func NewWorkflowExecuteJSONMessage(target string, jsonBytes []byte) Message {
	return Message{
		Target: target,
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
