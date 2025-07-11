// Package actor interfaces to access specific actor functionality without hard dependency to ergo/act or actors packages
package actor

// Handle agnostic interface of an Actor
type Handle interface {
	Send(to any, message any) error
}
