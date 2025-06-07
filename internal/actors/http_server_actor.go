package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/repos"
	"github.com/open-source-cloud/fuse/internal/workflow"
)

// HTTPServerActorName actor name
const HTTPServerActorName = "http_server"

// HTTPServerActorFactory redefines the HTTPServerActor factory type for better readability
type HTTPServerActorFactory Factory[*HTTPServerActor]

// NewHTTPServerActorFactory dependency injection function for creating the HTTPServerActor actor factory
func NewHTTPServerActorFactory(cfg *config.Config, graphFactory *workflow.GraphFactory, graphRepo repos.GraphRepo) *HTTPServerActorFactory {
	return &HTTPServerActorFactory{
		Factory: func() gen.ProcessBehavior {
			return &HTTPServerActor{
				config:       cfg,
				graphFactory: graphFactory,
				graphRepo:    graphRepo,
			}
		},
	}
}

// HTTPServerActor HTTP Server Actor
type HTTPServerActor struct {
	act.Actor
	config       *config.Config
	graphFactory *workflow.GraphFactory
	graphRepo    repos.GraphRepo
}

// Init (args ...any) called when the HTTPServerActor is being initialized
func (a *HTTPServerActor) Init(_ ...any) error {
	// get the gen.Log interface using Log method of embedded gen.Process interface
	a.Log().Debug("starting process %s", a.PID())

	metaBehavior := NewHTTPServerMeta(a.config, a.graphFactory, a.graphRepo)
	metaID, err := a.SpawnMeta(metaBehavior, gen.MetaOptions{})
	if err != nil {
		return err
	}
	a.Log().Debug("meta '%s' spawned with metaID: %s", HTTPServerMeta, metaID)

	return nil
}

// HandleMessage handles messages sent to HTTPServerActor
func (a *HTTPServerActor) HandleMessage(from gen.PID, message any) error {
	msg, ok := message.(messaging.Message)
	if !ok {
		a.Log().Error("message from %s is not a messaging.Message", from)
		return nil
	}
	a.Log().Info("got message from %s - %s", from, msg.Type)
	a.Log().Debug("args: %s", msg.Args)

	if msg.Type == messaging.TriggerWorkflow {
		err := a.Send(WorkflowSupervisorName, message)
		if err != nil {
			a.Log().Error("failed to send message to workflow supervisor: %s", err)
			return nil
		}
	}
	if msg.Type == messaging.AsyncFunctionResult {
		asyncFnResultMsg, _ := msg.AsyncFunctionResultMessage()
		err := a.Send(WorkflowHandlerName(asyncFnResultMsg.WorkflowID), message)
		if err != nil {
			a.Log().Error("failed to send message to workflow handler: %s", err)
		}
	}

	return nil
}

// Terminate called when HTTPServerActor is being terminated
func (a *HTTPServerActor) Terminate(reason error) {
	a.Log().Debug("%s terminated with reason: %s", a.PID(), reason)
}
