package workflow

type MessageType string

const (
	MessageStartWorkflow    MessageType = "workflow:start"
	MessageContinueWorkflow MessageType = "workflow:continue"
)

type Message interface {
	Type() MessageType
	Data() any
}

type message struct {
	msgType MessageType
	data    any
}

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
