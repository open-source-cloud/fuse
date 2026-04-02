package actors

import (
	"fmt"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/services"
)

// SchemaReplicationActorFactory builds the schema replication actor.
type SchemaReplicationActorFactory ActorFactory[*SchemaReplicationActor]

// NewSchemaReplicationActorFactory constructs the factory.
func NewSchemaReplicationActorFactory(
	cfg *config.Config,
	graphService services.GraphService,
) *SchemaReplicationActorFactory {
	return &SchemaReplicationActorFactory{
		Factory: func() gen.ProcessBehavior {
			return &SchemaReplicationActor{
				config:       cfg,
				graphService: graphService,
			}
		},
	}
}

// SchemaReplicationActor registers an ergo Event for local graph upserts and monitors peer events.
type SchemaReplicationActor struct {
	act.Actor

	config       *config.Config
	graphService services.GraphService

	eventToken gen.Ref
}

const graphSchemaUpsertEventName gen.Atom = "fuse_graph_schema_upsert"

// Init registers the local event (cluster mode) and subscribes to peer nodes' events.
func (a *SchemaReplicationActor) Init(_ ...any) error {
	if a.config == nil || !a.config.Cluster.Enabled {
		return nil
	}

	token, err := a.RegisterEvent(graphSchemaUpsertEventName, gen.EventOptions{
		Notify: false,
		Buffer: 16,
	})
	if err != nil {
		return fmt.Errorf("schema replication: RegisterEvent: %w", err)
	}
	a.eventToken = token

	local := a.Node().Name()
	for _, peer := range a.config.Cluster.PeerNodeNames() {
		peerAtom := gen.Atom(peer)
		if peerAtom == local {
			continue
		}
		target := gen.Event{Name: graphSchemaUpsertEventName, Node: peerAtom}
		buffered, subErr := a.MonitorEvent(target)
		if subErr != nil {
			a.Log().Warning("schema replication: MonitorEvent peer %s: %s", peer, subErr)
			continue
		}
		for _, ev := range buffered {
			if handleErr := a.handleReplicationEvent(ev); handleErr != nil {
				a.Log().Warning("schema replication: buffered event apply: %s", handleErr)
			}
		}
	}

	return nil
}

// HandleMessage receives local publish requests after HTTP upsert.
func (a *SchemaReplicationActor) HandleMessage(_ gen.PID, message any) error {
	msg, ok := message.(messaging.Message)
	if !ok {
		a.Log().Warning("schema replication: unexpected message type %T", message)
		return nil
	}
	if msg.Type != messaging.PublishGraphSchemaUpsert {
		return nil
	}
	payload, ok := msg.PublishGraphSchemaUpsertPayload()
	if !ok {
		a.Log().Warning("schema replication: bad publish payload")
		return nil
	}
	if !a.config.Cluster.Enabled || a.eventToken == (gen.Ref{}) {
		return nil
	}
	if err := a.SendEvent(graphSchemaUpsertEventName, a.eventToken, payload); err != nil {
		a.Log().Error("schema replication: SendEvent: %s", err)
	}
	return nil
}

// HandleEvent applies replicated schemas from remote nodes.
func (a *SchemaReplicationActor) HandleEvent(message gen.MessageEvent) error {
	switch message.Message.(type) {
	case gen.MessageEventStart, gen.MessageEventStop:
		return nil
	default:
		return a.handleReplicationEvent(message)
	}
}

func (a *SchemaReplicationActor) handleReplicationEvent(message gen.MessageEvent) error {
	payload, ok := message.Message.(messaging.GraphSchemaReplicationPayload)
	if !ok {
		a.Log().Debug("schema replication: skip non-payload event message %T", message.Message)
		return nil
	}
	if err := a.graphService.ApplyReplicatedUpsert(payload.SchemaID, payload.SchemaJSON); err != nil {
		a.Log().Error("schema replication: ApplyReplicatedUpsert %s: %s", payload.SchemaID, err)
		return nil
	}
	a.Log().Info("schema replication: applied schema %s from peer event", payload.SchemaID)
	return nil
}
