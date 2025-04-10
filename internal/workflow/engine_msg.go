package workflow

type EngineMessageType string

const (
	EngineMessageStartWorkflow EngineMessageType = "workflowWorker:start"
)

type EngineMessage interface {
	Type() EngineMessageType
	Data() interface{}
}

type engineMessage struct {
	msgType EngineMessageType
	data    interface{}
}

func NewEngineMessage(msgType EngineMessageType, data interface{}) EngineMessage {
	return &engineMessage{
		msgType: msgType,
		data:    data,
	}
}

func (m *engineMessage) Type() EngineMessageType {
	return m.msgType
}

func (m *engineMessage) Data() interface{} {
	return m.data
}
