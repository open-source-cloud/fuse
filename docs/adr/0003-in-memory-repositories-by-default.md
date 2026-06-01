# 0003. In-memory repositories by default, Postgres optional

- Status: Accepted
- Date: 2026-06-01
- Deciders: FUSE maintainers

> Backfill ADR: records a foundational decision made early in the project.

## Context and Problem Statement

The engine needs to store workflow schemas, in-flight workflow state, execution
journals, traces, packages, and (for HA) work claims. We wanted local development
and tests to be friction-free (no external services to stand up) while still
offering durable, shared persistence for production and high availability.

## Decision Drivers

- Zero-dependency local development and fast, isolated tests.
- A durable, shared backend for production and multi-node HA.
- A single seam so callers don't care which backend is active.
- Large payloads (graphs, journals, traces) shouldn't bloat the primary store.

## Considered Options

- **Repository interfaces with in-memory default + optional Postgres**, selected
  by config via DI; a separate pluggable object store for payloads.
- **Postgres always** — require a database for every run.
- **Embedded store** (SQLite/BoltDB) as the single default.

## Decision Outcome

Chosen option: **repository interfaces with `Memory*` implementations as the
default, and Postgres implementations selected when `DB_DRIVER=postgres`**, wired
by the DI layer. Memory repositories use `sync.RWMutex` for thread safety. Large
payloads go to a separate, pluggable **object store** (memory / filesystem / S3),
keeping the relational/primary store lean. HA builds on the Postgres path plus a
claims repository for work ownership.

### Consequences

- Good: `go test` and `make run` work with no external services; tests are isolated.
- Good: production gets durable, shared state by flipping config — no code changes.
- Good: payload storage is decoupled and independently swappable (memory/fs/s3).
- Bad: every repository has two implementations to keep in sync and test.
- Neutral: HA correctness depends on Postgres + the claims/lease mechanism, not the
  in-memory default.

## Pros and Cons of the Options

### Interfaces + memory default + optional Postgres

- Good: best of both worlds (dev simplicity, prod durability) behind one interface.
- Bad: duplicate implementations; risk of drift between them.

### Postgres always

- Good: one code path; production-like everywhere.
- Bad: external dependency for every test/dev run; slower, more setup.

### Embedded store (SQLite/BoltDB)

- Good: durable with no server.
- Bad: still file/IO setup for tests; weaker multi-node story; another query dialect.

## More Information

- Implementations: `internal/repositories/` (`*_memory.go`) and
  `internal/repositories/postgres/`; selection in `internal/app/di/repos.go`.
- Object store: `pkg/objectstore/` (memory / filesystem / S3) configured via
  `OBJECT_STORE_DRIVER`.
- Related: [ADR-0002](0002-ergo-actor-model-for-workflow-execution.md) (journal-based
  resume relies on these repositories).
