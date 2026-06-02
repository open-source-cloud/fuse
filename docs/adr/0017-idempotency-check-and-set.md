# 0017. Idempotent triggering via CheckAndSet with TTL

- Status: Accepted
- Date: 2026-06-01
- Deciders: FUSE maintainers

> Backfill ADR: records a foundational decision already implemented.

## Context and Problem Statement

Triggers ([ADR-0004](0004-multi-trigger-workflow-initiation.md)) can fire more than once for the
same logical event: a client retries an HTTP call, a cron tick fires on every HA node at once, or
the same event is observed by multiple nodes. Without deduplication this spawns duplicate
workflows. We need idempotent triggering that also works across HA nodes.

## Decision Drivers

- Deduplicate retried/duplicated triggers → at most one workflow per logical event.
- Work across multiple HA nodes (the dedup decision must be atomic and shared).
- Bounded storage (keys expire).
- Pluggable with the persistence model ([ADR-0003](0003-in-memory-repositories-by-default.md)).

## Considered Options

- **An idempotency store with an atomic `CheckAndSet` + TTL**, memory and Postgres backends.
- **Best-effort check-then-set** (separate read and write).
- **Database unique constraints** on a derived key, catching conflict errors.

## Decision Outcome

Chosen option: **an idempotency store keyed by an idempotency key, with an atomic `CheckAndSet`
and a TTL.** The `Store` interface (`internal/idempotency/store.go`) is
`Check`, `Set`, `Delete`, and `CheckAndSet(key, workflowID, ttl) (existingID, existed)`. The
memory backend guards `CheckAndSet` with a single write lock and purges expired entries
periodically; the Postgres backend (`internal/repositories/postgres/idempotency.go`) does it
atomically with `INSERT … ON CONFLICT DO NOTHING` + a CTE select, so concurrent HA nodes racing
on the same key produce exactly one winner. TTL is config-driven (`IDEMPOTENCY_TTL`, default 24h).

Usage by trigger type:
- **HTTP** — client-supplied `idempotencyKey`: `Check` on arrival returns the existing workflow
  id (response flagged `deduplicated`); `Set` after a successful trigger.
- **Cron** — deterministic key `cron:{schemaID}:{minuteBucket}` with `CheckAndSet` (TTL ~1h): all
  HA nodes fire together, only the first to set wins.
- **Event/webhook** — deterministic key derived from schema + event type + source + data hash via
  `CheckAndSet` (TTL ~10m).

### Consequences

- Good: at-most-once triggering per logical event, correct across HA nodes via atomic CheckAndSet.
- Good: bounded storage via TTL; backend follows the same memory/Postgres split as other repos.
- Bad: per-trigger TTLs (24h/1h/10m) are tuned constants, not yet centrally documented/configurable.
- Neutral: dedup is at the *trigger* boundary; idempotency of side effects inside a workflow is a
  separate concern (handled by journal replay, [ADR-0010](0010-durable-execution-journal-and-replay.md)).

## Pros and Cons of the Options

### Idempotency store with atomic CheckAndSet + TTL (chosen)

- Good: race-free across nodes; bounded; uniform across trigger types.
- Bad: scattered TTL constants.

### Best-effort check-then-set

- Good: simplest.
- Bad: racy under HA — two nodes can both pass the check and both spawn a workflow.

### DB unique constraint + conflict handling

- Good: leverages the database for atomicity.
- Bad: Postgres-only (no memory path); error-driven control flow; awkward TTL/expiry handling.

## More Information

- Code: `internal/idempotency/store.go`, `memory_store.go`;
  `internal/repositories/postgres/idempotency.go`; DI `internal/app/di/idempotency.go`;
  usage in `internal/handlers/trigger_workflow.go`, `internal/actors/cron_scheduler.go`,
  `event_trigger.go`. Config: `IdempotencyConfig` (`internal/app/config/config.go`).
- Related: [ADR-0004](0004-multi-trigger-workflow-initiation.md),
  [ADR-0018](0018-high-availability-and-clustering.md).
