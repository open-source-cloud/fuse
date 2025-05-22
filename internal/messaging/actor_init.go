// Package messaging ActorModel messaging types and helpers
package messaging

// NewActorInitMessage ActorInit message constructor
func NewActorInitMessage(args any) Message {
	return Message{
		Type: ActorInit,
		Args: args,
	}
}
