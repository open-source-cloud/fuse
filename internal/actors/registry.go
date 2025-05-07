package actors

import (
	"ergo.services/ergo/gen"
	"fmt"
)

func NewRegistry() *Registry {
	return &Registry{
		actors: make(map[string]gen.PID),
	}
}

type Registry struct {
	actors map[string]gen.PID
}

func (r *Registry) Register(actorName string, actor gen.PID) {
	r.actors[actorName] = actor
}

func (r *Registry) PIDof(actorName string) (gen.PID, error) {
	pid, ok := r.actors[actorName]
	if !ok {
		return gen.PID{}, fmt.Errorf("actor %s not found", actorName)
	}

	return pid, nil
}
