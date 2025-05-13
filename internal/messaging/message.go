package messaging

type MessageType string

const (
	ActorInit MessageType = "actor:init"
	TriggerWorkflow MessageType = "workflow:trigger"
	ExecuteFunction MessageType = "function:execute"
)

type Message struct {
	Type MessageType
	Args any
}
