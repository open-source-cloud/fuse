package workflow

// EngineMessageType type for engine messages
type EngineMessageType string

const (
	// EngineMessageStartWorkflow start a new workflow worker in engine
	EngineMessageStartWorkflow EngineMessageType = "workflowWorker:start"
)

// EngineMessage describes the engine message interface
type EngineMessage interface {
	Type() EngineMessageType
	Data() any
}

type engineMessage struct {
	msgType EngineMessageType
	data    any
}

// NewEngineMessage creates a new engine message
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
