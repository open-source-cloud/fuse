package messaging

func NewActorInitMessage(args any) Message {
	return Message{
		Type: ActorInit,
		Args: args,
	}
}
