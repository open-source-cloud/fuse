package services

import (
	"encoding/json"
	"sync"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/actors/actornames"
	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/rs/zerolog/log"
)

// SchemaUpsertPublisher notifies the cluster replication actor after a local schema upsert.
type SchemaUpsertPublisher interface {
	PublishLocalUpsert(schemaID string, schema *workflow.GraphSchema)
	BindNode(node gen.Node)
}

// ErgoSchemaUpsertPublisher sends PublishGraphSchemaUpsert to the schema replication actor (no-op until BindNode and when cluster is off).
type ErgoSchemaUpsertPublisher struct {
	cfg  *config.Config
	mu   sync.RWMutex
	node gen.Node
}

// NewErgoSchemaUpsertPublisher constructs a publisher; call BindNode from fx OnStart after the node exists.
func NewErgoSchemaUpsertPublisher(cfg *config.Config) *ErgoSchemaUpsertPublisher {
	return &ErgoSchemaUpsertPublisher{cfg: cfg}
}

// BindNode wires the ergo node for Send; safe to call once at startup.
func (p *ErgoSchemaUpsertPublisher) BindNode(node gen.Node) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.node = node
}

// PublishLocalUpsert marshals the schema and sends to the replication actor.
func (p *ErgoSchemaUpsertPublisher) PublishLocalUpsert(schemaID string, schema *workflow.GraphSchema) {
	if p == nil || p.cfg == nil || !p.cfg.Cluster.Enabled {
		return
	}
	p.mu.RLock()
	n := p.node
	p.mu.RUnlock()
	if n == nil {
		return
	}
	b, err := json.Marshal(schema)
	if err != nil {
		log.Error().Err(err).Str("schemaID", schemaID).Msg("schema replication: marshal failed")
		return
	}
	msg := messaging.NewPublishGraphSchemaUpsertMessage(messaging.GraphSchemaReplicationPayload{
		SchemaID:   schemaID,
		SchemaJSON: b,
	})
	if err := n.Send(gen.Atom(actornames.SchemaReplicationActorName), msg); err != nil {
		log.Warn().Err(err).Str("schemaID", schemaID).Msg("schema replication: send to actor failed")
	}
}
