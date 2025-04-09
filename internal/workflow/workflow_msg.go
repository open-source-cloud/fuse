package workflow

type MessageType string

const (
	MessageStartWorkflow MessageType = "workflow:start"
)

type Message interface {
	Type() MessageType
	Data() interface{}
}

type message struct {
	msgType MessageType
	data    interface{}
}

func NewMessage(msgType MessageType, data interface{}) Message {
	return &message{
		msgType: msgType,
		data:    data,
	}
}

func (m *message) Type() MessageType {
	return m.msgType
}

func (m *message) Data() interface{} {
	return m.data
}
