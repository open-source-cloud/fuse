package actors

import (
	"fmt"
	"os"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/actors/actornames"
	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// WorkflowClaimActorFactory builds the HA workflow claim actor.
type WorkflowClaimActorFactory ActorFactory[*WorkflowClaimActor]

// NewWorkflowClaimActorFactory constructs the factory.
func NewWorkflowClaimActorFactory(
	cfg *config.Config,
	claimRepo repositories.ClaimRepository,
) *WorkflowClaimActorFactory {
	return &WorkflowClaimActorFactory{
		Factory: func() gen.ProcessBehavior {
			return &WorkflowClaimActor{
				config:    cfg,
				claimRepo: claimRepo,
			}
		},
	}
}

// claimTickMsg is the periodic sweep tick message.
type claimTickMsg struct{}

// ClaimSweepNowMsg requests an immediate claim sweep (sent by PgListenerActor on NOTIFY).
type ClaimSweepNowMsg struct{}

// WorkflowClaimActor periodically claims unclaimed workflows and maintains heartbeats.
type WorkflowClaimActor struct {
	act.Actor

	config    *config.Config
	claimRepo repositories.ClaimRepository

	tickCancel gen.CancelFunc
}

// Init starts the periodic claim sweep timer.
func (a *WorkflowClaimActor) Init(_ ...any) error {
	a.Log().Info("starting workflow claim actor (node: %s, sweep: %s)",
		a.config.HA.NodeID, a.config.HA.ClaimSweepInterval)

	cancel, err := a.SendAfter(a.PID(), claimTickMsg{}, a.config.HA.ClaimSweepInterval)
	if err != nil {
		return fmt.Errorf("failed to schedule claim tick: %w", err)
	}
	a.tickCancel = cancel

	// Initial heartbeat
	a.heartbeat()

	return nil
}

// HandleMessage processes tick messages.
func (a *WorkflowClaimActor) HandleMessage(_ gen.PID, message any) error {
	switch message.(type) {
	case claimTickMsg:
		a.sweep()
		a.heartbeat()
		a.detectStaleNodes()

		// Reschedule the next tick
		cancel, err := a.SendAfter(a.PID(), claimTickMsg{}, a.config.HA.ClaimSweepInterval)
		if err != nil {
			a.Log().Error("failed to reschedule claim tick: %s", err)
		} else {
			a.tickCancel = cancel
		}
	case ClaimSweepNowMsg:
		// Fast path: PG LISTEN/NOTIFY triggered an immediate sweep
		a.sweep()
	default:
		a.Log().Warning("unknown message type: %T", message)
	}
	return nil
}

// Terminate is called on actor shutdown.
func (a *WorkflowClaimActor) Terminate(reason error) {
	a.Log().Info("workflow claim actor terminating: %s", reason)

	if a.tickCancel != nil {
		a.tickCancel()
	}

	// Release all workflows on graceful shutdown
	nodeID := a.nodeID()
	if releaseErr := a.claimRepo.ReleaseWorkflows(nodeID); releaseErr != nil {
		a.Log().Error("failed to release workflows on shutdown: %s", releaseErr)
	} else {
		a.Log().Info("released all claimed workflows for node %s", nodeID)
	}
}

func (a *WorkflowClaimActor) sweep() {
	nodeID := a.nodeID()
	claimed, err := a.claimRepo.ClaimWorkflows(nodeID, 10)
	if err != nil {
		a.Log().Error("claim sweep failed: %s", err)
		return
	}
	if len(claimed) == 0 {
		return
	}

	a.Log().Info("claimed %d workflow(s)", len(claimed))
	for _, cw := range claimed {
		triggerMsg := messaging.NewTriggerWorkflowMessage(cw.SchemaID, workflow.ID(cw.WorkflowID))
		if sendErr := a.Send(gen.Atom(actornames.WorkflowSupervisorName), triggerMsg); sendErr != nil {
			a.Log().Error("failed to send trigger for claimed workflow %s: %s", cw.WorkflowID, sendErr)
		}
	}
}

func (a *WorkflowClaimActor) heartbeat() {
	nodeID := a.nodeID()
	host, _ := os.Hostname()
	if err := a.claimRepo.Heartbeat(nodeID, host, 0); err != nil {
		a.Log().Error("heartbeat failed: %s", err)
	}
}

func (a *WorkflowClaimActor) detectStaleNodes() {
	stale, err := a.claimRepo.FindStaleNodes(a.config.HA.LeaseTimeout)
	if err != nil {
		a.Log().Error("stale node detection failed: %s", err)
		return
	}
	if len(stale) == 0 {
		return
	}

	a.Log().Warning("detected %d stale node(s): %v", len(stale), stale)
	released, reassignErr := a.claimRepo.ReassignFromStaleNodes(stale)
	if reassignErr != nil {
		a.Log().Error("failed to reassign from stale nodes: %s", reassignErr)
		return
	}
	if released > 0 {
		a.Log().Info("released %d workflow(s) from stale nodes", released)
	}
}

func (a *WorkflowClaimActor) nodeID() string {
	if a.config.HA.NodeID != "" {
		return a.config.HA.NodeID
	}
	hostname, _ := os.Hostname()
	return hostname
}
