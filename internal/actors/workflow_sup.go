package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/actors/actornames"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/repos"
	"github.com/open-source-cloud/fuse/internal/workflow"
)

// WorkflowSupervisorFactory redefines the WorkflowSupervisor supervisor actor factory type for better readability
type WorkflowSupervisorFactory ActorFactory[*WorkflowSupervisor]

// NewWorkflowSupervisorFactory a dependency injection that creates a new WorkflowSupervisor actor factory
func NewWorkflowSupervisorFactory(
	cfg *config.Config,
	workflowRepo repos.WorkflowRepo,
	workflowInstanceSup *WorkflowInstanceSupervisorFactory,
) *WorkflowSupervisorFactory {
	return &WorkflowSupervisorFactory{
		Factory: func() gen.ProcessBehavior {
			return &WorkflowSupervisor{
				config:              cfg,
				workflowRepo:        workflowRepo,
				workflowInstanceSup: workflowInstanceSup,
				workflowActors:      make(map[workflow.ID]gen.PID),
			}
		},
	}
}

// WorkflowSupervisor the WorkflowSupervisor supervisor actor
type WorkflowSupervisor struct {
	act.Supervisor

	config              *config.Config
	workflowRepo        repos.WorkflowRepo
	workflowInstanceSup *WorkflowInstanceSupervisorFactory

	workflowActors map[workflow.ID]gen.PID
}

// Init invoked on a spawn Supervisor process. This is a mandatory callback for the implementation
func (a *WorkflowSupervisor) Init(_ ...any) (act.SupervisorSpec, error) {
	a.Log().Debug("starting process %s", a.PID())

	// supervisor specification
	spec := act.SupervisorSpec{
		Type: act.SupervisorTypeSimpleOneForOne,
		// children
		Children: []act.SupervisorChildSpec{
			{
				Name:    actornames.WorkflowInstanceSupervisor,
				Factory: a.workflowInstanceSup.Factory,
			},
		},
		// strategy
		Restart: act.SupervisorRestart{
			Strategy:  act.SupervisorStrategyTransient,
			Intensity: 1, // How big bursts of restarts you want to tolerate.
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
		return nil
	}
	a.Log().Info("got message from %s - %s", from, msg.Type)
	a.Log().Debug("args: %s", msg.Args)

	if msg.Type == messaging.TriggerWorkflow {
		triggerMsg, err := msg.TriggerWorkflowMessage()
		if err != nil {
			a.Log().Error("failed to get trigger workflow message from message: %s", msg)
			return nil
		}
		err = a.spawnWorkflowActor(triggerMsg.SchemaID, true)
		if err != nil {
			a.Log().Error("failed to spawn workflow actor for schema id %s : %s", triggerMsg.SchemaID, err)
			return nil
		}
	}

	return nil
}

// Terminate invoked on a termination process
func (a *WorkflowSupervisor) Terminate(reason error) {
	a.Log().Info("process terminated with reason: %s", reason)
}

// HandleInspect invoked on the request made with gen.Process.Inspect(...)
func (a *WorkflowSupervisor) HandleInspect(from gen.PID, _ ...string) map[string]string {
	a.Log().Info("process got inspect request from %s", from)
	return nil
}

// HandleEvent handles events within a WorkflowSupervisor supervisor actor context
func (a *WorkflowSupervisor) HandleEvent(event gen.MessageEvent) error {
	a.Log().Info("received event %s with value: %#v", event.Event, event.Message)
	return nil
}

func (a *WorkflowSupervisor) spawnWorkflowActor(workflowOrSchemaID string, newWorkflow bool) error {
	var schemaID string
	var workflowID workflow.ID
	if newWorkflow {
		schemaID = workflowOrSchemaID
		workflowID = workflow.NewID()
	} else {
		existingWorkflow, err := a.workflowRepo.Get(workflowOrSchemaID)
		if err != nil {
			a.Log().Error("failed to get workflow %s: %s", workflowOrSchemaID, err)
			return err
		}
		workflowID = existingWorkflow.ID()
		schemaID = existingWorkflow.Schema().ID
	}
	err := a.StartChild(actornames.WorkflowInstanceSupervisor, workflowID, schemaID)
	if err != nil {
		a.Log().Error("failed to spawn child for schema id %s : %s", workflowOrSchemaID, err)
		return err
	}
	return nil
}
