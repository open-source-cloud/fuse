# 0019. Externalize large payloads to a pluggable object store

- Status: Accepted
- Date: 2026-06-01
- Deciders: FUSE maintainers

> Backfill ADR: records a foundational decision already implemented.

## Context and Problem Statement

Durable execution ([ADR-0010](0010-durable-execution-journal-and-replay.md)) and versioning
([ADR-0014](0014-schema-versioning-and-rollback.md)) produce large blobs: schema definitions,
journals, execution traces, and node inputs/outputs. Storing these inline in the primary
(relational) store bloats it and couples hot metadata to cold payloads. We need a place for
large payloads that scales independently and works from laptop to cloud.

## Decision Drivers

- Keep the primary store lean (metadata) and put large blobs elsewhere.
- One abstraction, several backends (zero-dep local dev → cloud in prod).
- Independent scaling/lifecycle for payloads.

## Considered Options

- **A pluggable `ObjectStore` interface (memory / filesystem / S3); DB rows hold metadata + a key.**
- **Store everything in Postgres** (large `bytea`/JSONB columns).
- **Filesystem only.**

## Decision Outcome

Chosen option: **a pluggable object store.** `pkg/objectstore` defines a minimal interface —
`Put(ctx,key,data)`, `Get`, `Delete` (idempotent), `Exists` — with three implementations chosen
by `OBJECT_STORE_DRIVER` (`internal/app/di/objectstore.go`): `memory` (default, dev/test),
`filesystem` (disk/NFS/PVC), and `s3` (S3-compatible via `minio-go`, works with AWS S3, MinIO,
rustfs, LocalStack; auto-creates the bucket). Config is `ObjectStoreConfig` (bucket, endpoint,
region, keys, SSL, key prefix).

Postgres repositories store **metadata in tables and the payload in the object store**, keyed by a
hierarchical path — e.g. traces at `workflows/{workflowID}/trace/{execID}/{input|output}.json`,
schemas at `schemas/{schemaID}/v{n}/definition.json`. The relational row references the object
key; the blob lives in the store.

### Consequences

- Good: primary store stays small; payloads scale independently and cheaply (S3); same code path
  from local dev (memory/fs) to prod (S3).
- Good: backend is swappable by config, no code change.
- Bad: a logical record spans two stores (row + object) → possible orphaned objects; no built-in
  GC/retention yet.
- Neutral: the memory/filesystem drivers are not durable/shared across nodes — HA uses S3 (or a
  shared volume) so all nodes see the same payloads.

## Pros and Cons of the Options

### Pluggable ObjectStore (chosen)

- Good: lean primary store; dev→prod parity; independent payload scaling.
- Bad: two-store consistency; orphan cleanup not yet automated.

### Everything in Postgres

- Good: single store, transactional with metadata.
- Bad: DB bloat and cost for large/cold blobs; backups/scaling get heavy.

### Filesystem only

- Good: simple, durable on one host.
- Bad: no shared access across HA nodes without network FS; no cloud-native scaling.

## More Information

- Code: `pkg/objectstore/store.go` (interface), `s3.go` (minio-go), memory/filesystem impls;
  `internal/app/di/objectstore.go`; payload keys in `internal/repositories/postgres/` (trace.go,
  graph.go, etc.). Config: `ObjectStoreConfig` (`internal/app/config/config.go`).
- Related: [ADR-0003](0003-in-memory-repositories-by-default.md),
  [ADR-0010](0010-durable-execution-journal-and-replay.md),
  [ADR-0018](0018-high-availability-and-clustering.md).
