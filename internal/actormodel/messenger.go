package actormodel

import "github.com/vladopajic/go-actor/actor"

type MessageReceiver string

// WorkflowEngine defines the workflow engine message receiver
const (
	AppSupervisor MessageReceiver = "supervisor"
	WorkflowEngine MessageReceiver = "engine"
	HttpServer MessageReceiver = "httpServer"
)

type SupervisorMessenger interface {
	Messenger
	SendMessageTo(receiver MessageReceiver, ctx actor.Context, msg Message)
}

type Messenger interface {
	SendMessage(ctx actor.Context, msg Message)
}
