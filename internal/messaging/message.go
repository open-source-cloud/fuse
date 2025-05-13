package messaging

type MessageType string

const (
	ActorInit MessageType = "actor:init"
	TriggerWorkflow MessageType = "workflow:trigger"
	ExecuteFunction MessageType = "function:execute"
	FunctionResult MessageType = "function:result"
)

type Message struct {
	Type MessageType
	Args any
}
