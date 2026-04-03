// Package actor interfaces to access specific actor functionality without hard dependency to ergo/act or actors packages
package actor

import "ergo.services/ergo/gen"

// Handle is the minimal surface for running package functions from a worker actor.
// Process.Send is only valid while the actor is handling a message; async completions
// must use Node().Send, which is safe from any goroutine.
type Handle interface {
	Send(to any, message any) error
	Node() gen.Node
}

// WorkflowHandlerPIDProvider is implemented by actors that know the workflow handler PID
// (e.g. WorkflowFunc) so internal async callbacks can use Node().Send(pid, ...) instead of
// routing by Atom name, which may not resolve for nested pool workers.
type WorkflowHandlerPIDProvider interface {
	WorkflowHandlerPID() gen.PID
}
