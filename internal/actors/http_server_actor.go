package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"fmt"
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/messaging"
)

const httpServerActorName = "http_server"

func NewHttpServerActorFactory(cfg *config.Config) *Factory[*HttpServerActor] {
	return &Factory[*HttpServerActor]{
		Name: httpServerActorName,
		Behavior: func() gen.ProcessBehavior {
			return &HttpServerActor{
				config:        cfg,
			}
		},
	}
}

type HttpServerActor struct {
	act.Actor
	config        *config.Config
	event         gen.Ref
}

// Init (args ...any)
func (a *HttpServerActor) Init(_ ...any) error {
	// get the gen.Log interface using Log method of embedded gen.Process interface
	a.Log().Info("starting process %s", a.PID())

	metaBehavior := NewHttpServerMeta(a.config)
	metaID, err := a.SpawnMeta(metaBehavior, gen.MetaOptions{})
	if err != nil {
		return err
	}
	a.Log().Info("meta '%s' spawned with metaID: %s", HttpServerMeta, metaID)

	return nil
}

func (a *HttpServerActor) HandleMessage(from gen.PID, message any) error {
	msg, ok := message.(messaging.Message)
	if !ok {
		a.Log().Error("message from %s is not a messaging.Message", from)
		return fmt.Errorf("message from %s is not a messaging.Message", from)
	}
	a.Log().Info("got message from %s - %s", from, msg.Type)

	switch msg.Type {
	case messaging.WorkflowExecuteJSON:
		err := a.Send(workflowSupervisorName, message)
		if err != nil {
			a.Log().Error("failed to send message to workflow supervisor: %s", err)
			return err
		}
		return nil
	}


	return nil
}

func (a *HttpServerActor) Terminate(reason error) {
	a.Log().Info("%s terminated with reason: %s", a.PID(), reason)
}
