package workflow

import "github.com/vladopajic/go-actor/actor"

type Context actor.Context

type Workflow interface {
	actor.Actor
	SendMessage(ctx Context, msg Message)
}
