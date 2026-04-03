package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/actors/actornames"
	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/repositories"
	internalworkflow "github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// WorkflowSupervisorFactory redefines the WorkflowSupervisor supervisor actor factory type for better readability
type WorkflowSupervisorFactory ActorFactory[*WorkflowSupervisor]

// NewWorkflowSupervisorFactory a dependency injection that creates a new WorkflowSupervisor actor factory
func NewWorkflowSupervisorFactory(
	cfg *config.Config,
	workflowRepository repositories.WorkflowRepository,
	workflowInstanceSup *WorkflowInstanceSupervisorFactory,
) *WorkflowSupervisorFactory {
	return &WorkflowSupervisorFactory{
		Factory: func() gen.ProcessBehavior {
			return &WorkflowSupervisor{
				config:              cfg,
				workflowRepository:  workflowRepository,
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
	workflowRepository  repositories.WorkflowRepository
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
		EnableHandleChild: true,
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

	switch msg.Type {
	case messaging.TriggerWorkflow:
		triggerMsg, err := msg.TriggerWorkflowMessage()
		if err != nil {
			a.Log().Error("failed to get trigger workflow message from message: %s", msg)
			return nil
		}
		err = a.spawnWorkflowActor(triggerMsg.SchemaID, triggerMsg.WorkflowID)
		if err != nil {
			a.Log().Error("failed to spawn workflow actor for schema id %s : %s", triggerMsg.SchemaID, err)
			return nil
		}
	case messaging.RecoverWorkflows:
		a.recoverWorkflows()
	case messaging.CancelWorkflow:
		cancelMsg, err := msg.CancelWorkflowMessage()
		if err != nil {
			a.Log().Error("failed to get cancel workflow message: %s", err)
			return nil
		}
		handlerName := actornames.WorkflowHandlerName(cancelMsg.WorkflowID)
		if sendErr := a.Send(gen.Atom(handlerName), message); sendErr != nil {
			a.Log().Warning("cancel requested for unknown/finished workflow %s: %s", cancelMsg.WorkflowID, sendErr)
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

// HandleChildStart invoked when a child process starts successfully
func (a *WorkflowSupervisor) HandleChildStart(name gen.Atom, pid gen.PID) error {
	a.Log().Info("child started: %s (pid: %s)", name, pid)
	return nil
}

// HandleChildTerminate invoked when a child process terminates.
// Cleans up the workflowActors map to free resources for completed workflows.
func (a *WorkflowSupervisor) HandleChildTerminate(name gen.Atom, pid gen.PID, reason error) error {
	a.Log().Info("child terminated: %s (pid: %s, reason: %s)", name, pid, reason)
	for wfID, wfPID := range a.workflowActors {
		if wfPID == pid {
			delete(a.workflowActors, wfID)
			a.Log().Info("cleaned up workflow actor for workflow %s", wfID)
			break
		}
	}
	return nil
}

// HandleEvent handles events within a WorkflowSupervisor supervisor actor context
func (a *WorkflowSupervisor) HandleEvent(event gen.MessageEvent) error {
	a.Log().Info("received event %s with value: %#v", event.Event, event.Message)
	return nil
}

func (a *WorkflowSupervisor) recoverWorkflows() {
	ids, err := a.workflowRepository.FindByState(internalworkflow.StateRunning, internalworkflow.StateSleeping)
	if err != nil {
		a.Log().Error("failed to query workflows for recovery: %s", err)
		return
	}
	if len(ids) == 0 {
		a.Log().Info("no workflows to recover")
		return
	}

	a.Log().Info("recovering %d workflow(s)", len(ids))
	for _, id := range ids {
		wf, getErr := a.workflowRepository.Get(id)
		if getErr != nil {
			a.Log().Error("failed to get workflow %s for recovery: %s", id, getErr)
			continue
		}
		schemaID := wf.Schema().ID
		if spawnErr := a.spawnWorkflowActor(schemaID, wf.ID()); spawnErr != nil {
			a.Log().Error("failed to recover workflow %s: %s", id, spawnErr)
		}
	}
}

func (a *WorkflowSupervisor) spawnWorkflowActor(schemaID string, workflowID workflow.ID) error {
	err := a.StartChild(actornames.WorkflowInstanceSupervisor, workflowID, schemaID)
	if err != nil {
		a.Log().Error("failed to spawn child for schema id %s : %s", schemaID, err)
		return err
	}

	// Find the PID of the newly started child to track it for cleanup
	expectedName := gen.Atom(actornames.WorkflowInstanceSupervisorName(workflowID))
	for _, child := range a.Children() {
		if child.Name == expectedName {
			a.workflowActors[workflowID] = child.PID
			a.Log().Debug("tracking workflow %s with pid %s", workflowID, child.PID)
			break
		}
	}
	return nil
}
