// Package actors ActorModel for the FUSE application
package actors

import (
	"ergo.services/ergo/gen"
)

// ActorFactory defines the factory type that all Actor Factories must implement
type ActorFactory[T gen.ProcessBehavior] struct {
	Factory func() gen.ProcessBehavior
}
