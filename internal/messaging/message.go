package messaging

// MessageType defines the MessageType type
type MessageType string

const (
	// ActorInit message type
	ActorInit       MessageType = "actor:init"
	// TriggerWorkflow message type
	TriggerWorkflow MessageType = "workflow:trigger"
	// ExecuteFunction message type
	ExecuteFunction MessageType = "function:execute"
	// FunctionResult message type
	FunctionResult      MessageType = "function:result"
	// AsyncFunctionResult message type
	AsyncFunctionResult MessageType = "function:async:result"
)

// Message defines the basic Message
type Message struct {
	Type MessageType
	Args any
}
