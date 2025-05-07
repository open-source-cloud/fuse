package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
)

type Factory[T gen.ProcessBehavior] struct {
	Name     string
	Behavior func() gen.ProcessBehavior
}

func (f *Factory[T]) ApplicationMemberSpec() gen.ApplicationMemberSpec {
	return gen.ApplicationMemberSpec{
		Name:    gen.Atom(f.Name),
		Factory: f.Behavior,
	}
}

func (f *Factory[T]) SupervisorChildSpec() act.SupervisorChildSpec {
	return act.SupervisorChildSpec{
		Name:    gen.Atom(f.Name),
		Factory: f.Behavior,
	}
}
