package actors

import (
	"ergo.services/ergo/gen"
)

type Factory[T gen.ProcessBehavior] struct {
	Factory func() gen.ProcessBehavior
}
