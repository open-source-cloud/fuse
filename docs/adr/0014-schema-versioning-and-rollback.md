# 0014. Immutable schema versioning with an active-version pointer

- Status: Accepted
- Date: 2026-06-01
- Deciders: FUSE maintainers

> Backfill ADR: records a foundational decision already implemented.

## Context and Problem Statement

Workflow schemas ([ADR-0013](0013-workflow-schema-model-and-input-mapping.md)) change over time,
but workflows are long-running — an in-flight instance must keep executing against the definition
it started with, while new triggers should use the latest approved definition. We need a
versioning model that supports safe iteration, rollback, and history without mutating a schema
that running workflows depend on.

## Decision Drivers

- Never mutate a schema in place (in-flight workflows depend on it).
- Atomically switch which version new executions use.
- Keep full history; support rollback and inspection.

## Considered Options

- **Immutable versions + an active-version pointer**, with rollback as a new version.
- **Overwrite the schema in place** on each update.
- **Git-style content-addressed schemas** (hash-identified, no explicit version numbers).

## Decision Outcome

Chosen option: **immutable versions with an active pointer.** Every `Upsert` creates a new
`SchemaVersion` (`internal/workflow/versioned_schema.go`: `SchemaID`, `Version`, cloned
`Schema`, `CreatedAt`, `Comment`, `IsActive`) — it never overwrites. The repository keeps, per
schema id, the full `versions` slice and an `activeVersions` pointer
(`internal/repositories/graph_memory.go`; Postgres equivalent). The new version becomes active
and the previous one is marked inactive; `FindByID` returns the graph for the active version.

Service operations (`internal/services/graph_service.go`):
- `Upsert` → append version N (active).
- `ListVersions` / `GetVersionHistory` → history + `{activeVersion, latestVersion, totalVersions}`.
- `FindByIDAndVersion` → read any past version.
- `SetActiveVersion` → repoint the active version atomically.
- `Rollback(toVersion)` → clone an old version's content into a **new** version (N+1) and
  activate it — rollback is forward-only, preserving history.

REST: `PUT /v1/schemas/{id}`, `GET /v1/schemas/{id}/versions`,
`POST /v1/schemas/{id}/versions/{v}/activate`, `POST /v1/schemas/{id}/rollback`.

### Consequences

- Good: in-flight workflows are unaffected by new versions; deployments flip versions atomically;
  full audit/rollback.
- Good: rollback-as-new-version keeps history linear and avoids destructive reverts.
- Bad: storage grows with every upsert (mitigated by payload externalization,
  [ADR-0019](0019-object-store-payload-externalization.md)); no automatic pruning/retention yet.
- Neutral: callers choosing a specific version must pass it explicitly; default is the active one.

## Pros and Cons of the Options

### Immutable versions + active pointer (chosen)

- Good: safe iteration, atomic switch, rollback, history.
- Bad: unbounded version growth without retention.

### Overwrite in place

- Good: trivial, minimal storage.
- Bad: breaks in-flight workflows; no history or rollback.

### Content-addressed (hash) schemas

- Good: natural dedup and immutability.
- Bad: no human-friendly ordering; "active" still needs a pointer; bigger change for little gain
  over numbered versions.

## More Information

- Code: `internal/workflow/versioned_schema.go`; `internal/services/graph_service.go`
  (Upsert/ListVersions/SetActiveVersion/Rollback/GetVersionHistory);
  `internal/repositories/graph_memory.go` (+ `postgres/graph.go`).
- Related: [ADR-0013](0013-workflow-schema-model-and-input-mapping.md),
  [ADR-0003](0003-in-memory-repositories-by-default.md),
  [ADR-0018](0018-high-availability-and-clustering.md) (versions replicate across nodes).
