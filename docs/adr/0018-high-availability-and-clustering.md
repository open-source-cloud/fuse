# 0018. High availability: claims, clustering, and schema replication

- Status: Accepted
- Date: 2026-06-01
- Deciders: FUSE maintainers

> Backfill ADR: records a foundational decision already implemented.

## Context and Problem Statement

For production, FUSE runs as multiple nodes so it survives a node failure and scales out. That
requires: distributing workflow ownership across nodes (and recovering a dead node's work),
nodes discovering each other for actor messaging, and keeping workflow schemas consistent
cluster-wide. We want this without standing up a heavyweight consensus system.

## Decision Drivers

- No workflow lost or run twice when a node dies (failover).
- Dynamic node discovery (incl. Kubernetes autoscaling).
- Cluster-wide schema consistency.
- Reuse existing infra (Postgres, ergo) over adding Raft/ZooKeeper.

## Considered Options

- **DB lease-based claims + ergo clustering (static or etcd discovery) + Postgres LISTEN/NOTIFY +
  ergo-Event schema replication.**
- **A consensus system** (Raft/etcd-as-coordinator) owning scheduling.
- **External queue** distributing work to stateless workers.

## Decision Outcome

Chosen option: **lease-based DB claims + ergo clustering + LISTEN/NOTIFY, composed from existing
infrastructure.** Three mechanisms:

1. **Work claiming (failover).** `ClaimRepository` (Postgres) lets each node atomically claim
   unclaimed or lease-expired workflows with `UPDATE … FOR UPDATE SKIP LOCKED`. A
   `WorkflowClaimActor` periodically sweeps (`HA_CLAIM_SWEEP_INTERVAL`, default 5s), heartbeats
   (`node_heartbeats`), and reassigns workflows from nodes whose heartbeat is older than
   `HA_LEASE_TIMEOUT` (default 30s). The memory backend is a no-op (HA off).

2. **Lower-latency claims.** A `PgListenerActor` subscribes to Postgres **LISTEN/NOTIFY**
   (`workflow_state_change`) on a dedicated connection; a newly available workflow nudges the
   claim actor (`ClaimSweepNowMsg`) to claim immediately instead of waiting for the next sweep.

3. **Discovery + replication.** ergo clustering is enabled by `CLUSTER_ENABLED` with a shared
   `cookie` and acceptor port; discovery is **static** (`CLUSTER_PEER_NODES`) or **etcd**
   (`ergo.services/registrar/etcd`, lease-based, for dynamic peers/autoscaling). Schema upserts
   propagate via an ergo **Event** (`fuse_graph_schema_upsert`): the `SchemaReplicationActor`
   monitors peers (static list or etcd-discovered) and applies replicated upserts idempotently,
   so [ADR-0014](0014-schema-versioning-and-rollback.md) versions stay consistent across nodes.

### Consequences

- Good: node failure self-heals (lease expiry → reclaim); no extra coordinator to operate;
  builds on Postgres + ergo already in the stack; etcd path supports HPA.
- Good: LISTEN/NOTIFY gives near-real-time claiming without tight polling.
- Bad: HA correctness requires Postgres ([ADR-0003](0003-in-memory-repositories-by-default.md)) —
  the memory backend is single-node only.
- Bad: schema replication is best-effort ergo-Event fan-out (no quorum) — eventual, not strongly
  consistent; concurrency limits are per-node ([ADR-0016](0016-concurrency-and-rate-limiting.md)).
- Neutral: failover granularity/latency is bounded by the sweep interval and lease timeout.

## Pros and Cons of the Options

### DB claims + ergo + LISTEN/NOTIFY (chosen)

- Good: reuses existing infra; self-healing; dynamic discovery; low operational burden.
- Bad: Postgres-dependent; eventual schema consistency; per-node limits.

### Consensus system owning scheduling

- Good: strong consistency, principled leader/partition handling.
- Bad: heavy to run and reason about; large addition for the gains needed here.

### External queue + stateless workers

- Good: mature horizontal scaling.
- Bad: doesn't fit stateful, resumable, per-instance workflow execution; another system to run.

## More Information

- Code: `internal/repositories/claim.go` (+ `postgres/claim.go`),
  `internal/actors/workflow_claim_actor.go`, `internal/actors/pg_listener_actor.go`,
  `internal/repositories/postgres/listener.go`, `internal/actors/schema_replication_actor.go`,
  `internal/app/fuse.go` (ergo cluster + etcd registrar). Config: `HAConfig`, `ClusterConfig`
  (`internal/app/config/config.go`).
- Related: [ADR-0002](0002-ergo-actor-model-for-workflow-execution.md),
  [ADR-0003](0003-in-memory-repositories-by-default.md),
  [ADR-0017](0017-idempotency-check-and-set.md), [ADR-0014](0014-schema-versioning-and-rollback.md).
