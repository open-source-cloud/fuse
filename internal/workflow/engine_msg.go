package workflow

type EngineMessageType string

const (
	EngineMessageStartWorkflow EngineMessageType = "workflowWorker:start"
)

type EngineMessage interface {
	Type() EngineMessageType
	Data() any
}

type engineMessage struct {
	msgType EngineMessageType
	data    any
}

func NewEngineMessage(msgType EngineMessageType, data any) EngineMessage {
	return &engineMessage{
		msgType: msgType,
		data:    data,
	}
}

func (m *engineMessage) Type() EngineMessageType {
	return m.msgType
}

func (m *engineMessage) Data() any {
	return m.data
}
