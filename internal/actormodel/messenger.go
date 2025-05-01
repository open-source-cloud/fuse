// Package actormodel actor model helpers
package actormodel

import "github.com/vladopajic/go-actor/actor"

// MessageReceiver defines a message receiver enum
type MessageReceiver string

// WorkflowEngine defines the workflow engine message receiver
const (
	// AppSupervisor app supervisor
	AppSupervisor MessageReceiver = "supervisor"
	// WorkflowEngine workflow engine
	WorkflowEngine MessageReceiver = "engine"
	// HTTPServer http server
	HTTPServer MessageReceiver = "httpServer"
)

// SupervisorMessenger defines a messenger that is also a supervisor
type SupervisorMessenger interface {
	Messenger
	SendMessageTo(receiver MessageReceiver, ctx actor.Context, msg Message)
}

// Messenger defines a messenger
type Messenger interface {
	SendMessage(ctx actor.Context, msg Message)
}
