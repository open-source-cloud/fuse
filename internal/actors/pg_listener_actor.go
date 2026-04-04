package actors

import (
	"context"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/actors/actornames"
	"github.com/open-source-cloud/fuse/internal/repositories/postgres"
)

// PgListenerActorFactory builds the PG LISTEN/NOTIFY actor.
type PgListenerActorFactory ActorFactory[*PgListenerActor]

// NewPgListenerActorFactory constructs the factory.
func NewPgListenerActorFactory(
	listener *postgres.PgListener,
) *PgListenerActorFactory {
	return &PgListenerActorFactory{
		Factory: func() gen.ProcessBehavior {
			return &PgListenerActor{
				listener: listener,
			}
		},
	}
}

// PgListenerActor subscribes to PG LISTEN/NOTIFY and routes workflow state changes
// to the WorkflowClaimActor for immediate claim attempts.
type PgListenerActor struct {
	act.Actor

	listener *postgres.PgListener
	cancel   context.CancelFunc
}

// Init starts the background listener goroutine.
func (a *PgListenerActor) Init(_ ...any) error {
	a.Log().Info("starting PG listener actor")

	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel

	notifyCh := make(chan postgres.WorkflowNotification, 64)

	// Background goroutine: reads from PG LISTEN
	go func() {
		if err := a.listener.Listen(ctx, notifyCh); err != nil {
			a.Log().Error("PG listener error: %s", err)
		}
	}()

	// Background goroutine: reads from channel and sends to claim actor
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case wn, ok := <-notifyCh:
				if !ok {
					return
				}
				// Only trigger immediate sweep for unclaimed workflows
				if wn.ClaimedBy == "" && (wn.State == "untriggered" || wn.State == "running" || wn.State == "sleeping") {
					if err := a.Send(gen.Atom(actornames.WorkflowClaimActorName), ClaimSweepNowMsg{}); err != nil {
						a.Log().Warning("failed to send sweep-now to claim actor: %s", err)
					}
				}
			}
		}
	}()

	return nil
}

// HandleMessage is unused — notifications come via the background goroutine.
func (a *PgListenerActor) HandleMessage(_ gen.PID, _ any) error {
	return nil
}

// Terminate cleans up the listener connection.
func (a *PgListenerActor) Terminate(reason error) {
	a.Log().Info("PG listener actor terminating: %s", reason)
	if a.cancel != nil {
		a.cancel()
	}
	if a.listener != nil {
		if err := a.listener.Close(context.Background()); err != nil {
			a.Log().Error("failed to close PG listener: %s", err)
		}
	}
}
