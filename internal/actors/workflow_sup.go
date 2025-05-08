package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/expr-lang/expr/types"
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/graph"
	"github.com/open-source-cloud/fuse/internal/graph/schema"
	"github.com/open-source-cloud/fuse/internal/messaging"
)

const workflowSupervisorName = "workflow_supervisor"

func NewWorkflowSupervisorFactory(
	cfg *config.Config,
	actorRegistry *Registry,
	workflowActorFactory *Factory[*WorkflowActor],
) *Factory[*WorkflowSupervisor] {
	return &Factory[*WorkflowSupervisor]{
		Name: workflowSupervisorName,
		Behavior: func() gen.ProcessBehavior {
			return &WorkflowSupervisor{
				config:               cfg,
				actorRegistry:        actorRegistry,
				workflowActorFactory: workflowActorFactory,
			}
		},
	}
}

type WorkflowSupervisor struct {
	act.Supervisor
	config               *config.Config
	actorRegistry        *Registry
	workflowActorFactory *Factory[*WorkflowActor]
}

// Init invoked on a spawn Supervisor process. This is a mandatory callback for the implementation
func (a *WorkflowSupervisor) Init(args ...any) (act.SupervisorSpec, error) {
	a.Log().Info("starting process %s", a.PID())
	var spec act.SupervisorSpec

	// set supervisor type
	spec.Type = act.SupervisorTypeSimpleOneForOne

	// add children
	spec.Children = []act.SupervisorChildSpec{
		a.workflowActorFactory.SupervisorChildSpec(),
	}

	// set strategy
	spec.DisableAutoShutdown = true
	spec.Restart.Strategy = act.SupervisorStrategyTransient
	spec.Restart.Intensity = 0 // How big bursts of restarts you want to tolerate.
	spec.Restart.Period = 5    // In seconds.

	a.actorRegistry.Register(workflowSupervisorName, a.PID())

	return spec, nil
}

// HandleMessage invoked if Supervisor received a message sent with gen.Process.Send(...).
// Non-nil value of the returning error will cause termination of this process.
// To stop this process normally, return gen.TerminateReasonNormal or
// gen.TerminateReasonShutdown. Any other - for abnormal termination.
func (a *WorkflowSupervisor) HandleMessage(from gen.PID, message any) error {
	a.Log().Info("got message from %s:%s", from, types.TypeOf(message))

	msg, ok := message.(messaging.Message)
	if !ok {
		a.Log().Error("message is not a messaging.Message")
		return nil
	}
	jsonMsg, err := msg.WorkflowExecuteJSONMessage()
	if err != nil {
		a.Log().Error("failed to get json message: %s", err)
		return nil
	}
	schemaRef, err := schema.FromJSON(jsonMsg.JsonBytes)
	if err != nil {
		a.Log().Error("failed to parse schema: %s", err)
		return nil
	}
	_, err = graph.NewGraphFromSchema(schemaRef)
	if err != nil {
		a.Log().Error("failed to create graph from schema: %s", err)
		return nil
	}

	return nil
}

// Terminate invoked on a termination process
func (a *WorkflowSupervisor) Terminate(reason error) {
	a.Log().Info("process terminated with reason: %s", reason)
}

// HandleInspect invoked on the request made with gen.Process.Inspect(...)
func (a *WorkflowSupervisor) HandleInspect(from gen.PID, item ...string) map[string]string {
	a.Log().Info("process got inspect request from %s", from)
	return nil
}

func (a *WorkflowSupervisor) HandleEvent(event gen.MessageEvent) error {
	a.Log().Info("received event %s with value: %#v", event.Event, event.Message)
	return nil
}
