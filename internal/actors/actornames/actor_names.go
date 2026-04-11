// Package actornames defines the constants and helper functions for Actor names
package actornames

// WorkflowSupervisorName WorkflowSupervisor supervisor actor name
const WorkflowSupervisorName = "workflow_sup"

// MuxServerName is the name of the MuxServer actor
const MuxServerName = "mux_server"

// MuxServerSupName is the name of the MuxServerSup actor
const MuxServerSupName = "mux_server_sup"

// WorkflowInstanceSupervisor is the name of the WorkflowInstanceSup actor
const WorkflowInstanceSupervisor = "workflow_instance_sup"

// SchemaReplicationActorName is the schema cluster-replication actor (ergo Events producer/consumer).
const SchemaReplicationActorName = "schema_replication"

// FuseApplicationName is the ergo application name (see app.Fuse.Load). Used for etcd ResolveApplication peers.
const FuseApplicationName = "fuse_app"

// WorkflowClaimActorName is the HA workflow claim actor name.
const WorkflowClaimActorName = "workflow_claim"

// PgListenerActorName is the PG LISTEN/NOTIFY actor name.
const PgListenerActorName = "pg_listener"

// CronSchedulerName is the cron scheduler actor name.
const CronSchedulerName = "cron_scheduler"

// WebhookRouterName is the webhook router actor name.
const WebhookRouterName = "webhook_router"

// EventTriggerName is the event trigger actor name.
const EventTriggerName = "event_trigger"
