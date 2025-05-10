package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"fmt"
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/workflow"
)

const WorkflowSupervisorName = "workflow_sup"

type WorkflowSupervisorFactory Factory[*WorkflowSupervisor]

func NewWorkflowSupervisorFactory(
	cfg *config.Config,
	workflowHandler *WorkflowHandlerFactory,
) *WorkflowSupervisorFactory {
	return &WorkflowSupervisorFactory{
		Factory: func() gen.ProcessBehavior {
			return &WorkflowSupervisor{
				config:          cfg,
				workflowHandler: workflowHandler,
				workflowActors:  make(map[workflow.ID]gen.PID),
			}
		},
	}
}

type WorkflowSupervisor struct {
	act.Supervisor

	config          *config.Config
	workflowHandler *WorkflowHandlerFactory

	workflowActors map[workflow.ID]gen.PID
}

// Init invoked on a spawn Supervisor process. This is a mandatory callback for the implementation
func (a *WorkflowSupervisor) Init(_ ...any) (act.SupervisorSpec, error) {
	a.Log().Info("starting process %s", a.PID())

	// supervisor specification
	spec := act.SupervisorSpec{
		Type: act.SupervisorTypeSimpleOneForOne,
		// children
		Children: []act.SupervisorChildSpec{
			{
				Name: "workflow_handler",
				Factory: a.workflowHandler.Factory,
			},
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

	//switch msg.Type {
	//case messaging.ChildInit:
	//	workflowID, ok := msg.Data.(workflow.ID)
	//	if !ok {
	//		a.Log().Error("failed to get workflowID from message: %s", msg)
	//		return fmt.Errorf("failed to get workflowID from message: %s", msg)
	//	}
	//	a.Log().Info("got child init message from %s for workflowID %s", from, workflowID)
	//	a.workflowActors[workflowID] = from
	//case messaging.TriggerWorkflow:
	//	triggerMsg, err := msg.TriggerWorkflowMessage()
	//	if err != nil {
	//		a.Log().Error("failed to get trigger workflow message from message: %s", msg)
	//		return fmt.Errorf("failed to get trigger workflow message from message: %s", msg)
	//	}
	//	err = a.spawnWorkflowActor(triggerMsg.SchemaID)
	//	if err != nil {
	//		return err
	//	}
	//}

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

func (a *WorkflowSupervisor) spawnWorkflowActor(schemaID string) error {
	//err := a.StartChild(gen.Atom(a.workflowHandler.Name), schemaID)
	//if err != nil {
	//	a.Log().Error("failed to spawn child for schema ID %s : %s", schemaID, err)
	//	return err
	//}
	return nil
}
