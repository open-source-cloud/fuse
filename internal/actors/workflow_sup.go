package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"fmt"
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/graph"
	"github.com/open-source-cloud/fuse/internal/graph/schema"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/workflow"
)

const workflowSupervisorName = "workflow_supervisor"

func NewWorkflowSupervisorFactory(
	cfg *config.Config,
	schemaRepo workflow.SchemaRepo,
	workflowActorFactory *Factory[*WorkflowActor],
) *Factory[*WorkflowSupervisor] {
	return &Factory[*WorkflowSupervisor]{
		Name: workflowSupervisorName,
		Behavior: func() gen.ProcessBehavior {
			return &WorkflowSupervisor{
				config:               cfg,
				schemaRepo:           schemaRepo,
				workflowActorFactory: workflowActorFactory,
				workflowActors:       make(map[string]gen.PID),
			}
		},
	}
}

type WorkflowSupervisor struct {
	act.Supervisor

	config               *config.Config
	schemaRepo           workflow.SchemaRepo
	workflowActorFactory *Factory[*WorkflowActor]

	workflowActors map[string]gen.PID
}

// Init invoked on a spawn Supervisor process. This is a mandatory callback for the implementation
func (a *WorkflowSupervisor) Init(args ...any) (act.SupervisorSpec, error) {
	a.Log().Info("starting process %s", a.PID())

	// supervisor specification
	spec := act.SupervisorSpec{
		Type: act.SupervisorTypeSimpleOneForOne,
		// children
		Children: []act.SupervisorChildSpec{
			a.workflowActorFactory.SupervisorChildSpec(),
		},
		// strategy
		Restart: act.SupervisorRestart{
			Strategy:  act.SupervisorStrategyTransient,
			Intensity: 5, // How big bursts of restarts you want to tolerate.
			Period:    5, // In seconds.
		},
	}

	return spec, nil
}

// HandleMessage invoked if Supervisor received a message sent with gen.Process.Send(...).
// Non-nil value of the returning error will cause termination of this process.
// To stop this process normally, return gen.TerminateReasonNormal or
// gen.TerminateReasonShutdown. Any other - for abnormal termination.
func (a *WorkflowSupervisor) HandleMessage(from gen.PID, message any) error {
	msg, ok := message.(messaging.Message)
	if !ok {
		a.Log().Error("message from %s is not a messaging.Message", from)
		return fmt.Errorf("message from %s is not a messaging.Message", from)
	}
	a.Log().Info("got message from %s - %s", from, msg.Type)

	switch msg.Type {
	case messaging.WorkflowExecuteJSON:
		jsonMsg, err := msg.WorkflowExecuteJSONMessage()
		if err != nil {
			a.Log().Error("failed to get json message: %s", err)
			return nil
		}
		err = a.spawnWorkflowActor(jsonMsg.JsonBytes)
		if err != nil {
			a.Log().Error("failed to spawn workflow actor: %s", err)
			return err
		}
	case messaging.ChildInit:
		schemaID, ok := msg.Data.(string)
		if !ok {
			a.Log().Error("failed to get schema ID from message: %s", msg)
			return fmt.Errorf("failed to get schema ID from message: %s", msg)
		}
		a.Log().Info("got child init message from %s for schema ID %s", from, schemaID)
		a.workflowActors[schemaID] = from
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

func (a *WorkflowSupervisor) spawnWorkflowActor(jsonBytes []byte) error {
	schemaRef, err := schema.FromJSON(jsonBytes)
	if err != nil {
		a.Log().Error("failed to parse schema: %s", err)
		return err
	}
	graphRef, err := graph.NewGraphFromSchema(schemaRef)
	if err != nil {
		a.Log().Error("failed to create graph from schema: %s", err)
		return err
	}

	workflowSchema := workflow.NewSchema(schemaRef.ID, graphRef)
	a.schemaRepo.Save(schemaRef.ID, workflowSchema)

	err = a.StartChild(gen.Atom(a.workflowActorFactory.Name), schemaRef.ID)
	if err != nil {
		a.Log().Error("failed to spawn child: %s for schema ID %s", err, schemaRef.ID)
		return err
	}
	return nil
}
