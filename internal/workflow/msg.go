package workflow

// MessageType go type for message type parameters
type MessageType string

const (
	// MessageStartWorkflow message that starts a workflow execution
	MessageStartWorkflow MessageType = "workflow:start"
	// MessageContinueWorkflow message that continues a workflow execution
	MessageContinueWorkflow MessageType = "workflow:continue"
)

// Message defines a message interface
type Message interface {
	Type() MessageType
	Data() any
}

type message struct {
	msgType MessageType
	data    any
}

// NewMessage returns a new Message object with type and data
func NewMessage(msgType MessageType, data any) Message {
	return &message{
		msgType: msgType,
		data:    data,
	}
}

func (m *message) Type() MessageType {
	return m.msgType
}

func (m *message) Data() any {
	return m.data
}
