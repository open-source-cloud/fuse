# 0010. Durable execution via an append-only journal

- Status: Accepted
- Date: 2026-06-01
- Deciders: FUSE maintainers

> Backfill ADR: records a foundational decision already implemented.

## Context and Problem Statement

FUSE is a "Stateful Events" engine: workflows are long-running, can sleep, wait on external
systems, fan out, and must survive process crashes and (in HA) node failures. The engine
needs a way to make an in-flight workflow's progress durable and to resume it deterministically
after a restart without re-running already-completed steps.

## Decision Drivers

- Crash/restart recovery without duplicate side effects (don't re-run completed nodes).
- Support long-running primitives: sleep, external async, sub-workflows, ForEach.
- Deterministic replay; auditable execution history.
- Works with the actor model ([ADR-0002](0002-ergo-actor-model-for-workflow-execution.md)) and
  pluggable persistence ([ADR-0003](0003-in-memory-repositories-by-default.md)).

## Considered Options

- **Append-only journal of every step transition + replay on recovery** (event-sourcing style).
- **Snapshot current state periodically** and restore the latest snapshot.
- **No durability** — keep workflow state in memory only.

## Decision Outcome

Chosen option: **an append-only journal**. Every transition is recorded as an immutable
`JournalEntry` with a monotonic `Sequence` and `Timestamp` (`internal/workflow/journal.go`).
Entry types cover the full lifecycle: `step:started|completed|failed|retrying|manual-retry`,
`thread:created|finished`, `state:changed`, `sleep:started|completed`,
`awakeable:created|resolved`, `subworkflow:started|completed`, and
`foreach:started|iteration:started|iteration:completed|completed`.

`Journal.Append` assigns the sequence; `NewEntries()`/`MarkPersisted()` track what has been
flushed so `lastPersisted` avoids re-writing. On startup, `WorkflowHandler.Init` loads the
journal (`JournalRepository.LoadAll`), `Workflow.Resume()` replays entries to rebuild threads,
audit log, and aggregated output, identifies pending (started-but-not-completed) work, and
returns the next `Action`. Untriggered workflows call `Trigger()` instead.

Two complementary async primitives sit on top of the journal:
- **Async function results** — a function returns `NewFunctionResultAsync()` and later calls
  `execInfo.Finish(...)`, delivered back as an internal actor message (in-process, e.g. timer,
  `ai/chat`). See [ADR-0005](0005-ai-agents-as-workflow-nodes-phased-roadmap.md).
- **Awakeables** — durable promises (`internal/workflow/awakeable.go`): an `Awakeable`
  (`ID`, `Status` pending/resolved/timed_out/cancelled, `Timeout`, `DeadlineAt`, `Result`) is
  persisted and resolved by an **external** caller via `POST /v1/awakeables/{id}/resolve`,
  surviving restarts. Use awakeables for external/human-in-the-loop waits; use the Finish
  callback for in-process async.
- **Sub-workflows** (`internal/workflow/subworkflow.go`): a `SubWorkflowRef` links parent
  thread/exec to a child workflow; `async=false` makes the parent wait for
  `SubWorkflowCompleted`, `async=true` lets it continue. Journaled as `subworkflow:started|completed`.

### Consequences

- Good: deterministic resume after crash/restart; complete audit trail; long-running primitives
  fall out of the same mechanism.
- Good: append-only + sequence numbers make persistence incremental and idempotent.
- Bad: every transition is a write; the journal grows per workflow (mitigated by externalizing
  payloads — [ADR-0019](0019-object-store-payload-externalization.md)).
- Neutral: replay correctness depends on entries being deterministic and complete.

## Pros and Cons of the Options

### Append-only journal + replay

- Good: full history, deterministic recovery, natural fit for sleep/await/sub-workflows.
- Bad: more writes; replay cost grows with history length.

### Periodic snapshots

- Good: fast restore, less storage churn.
- Bad: work since the last snapshot is lost or must be re-run; weaker audit trail.

### In-memory only

- Good: simplest, fastest.
- Bad: any crash loses all in-flight workflows — unacceptable for a durable engine.

## More Information

- Code: `internal/workflow/journal.go`, `awakeable.go`, `subworkflow.go`;
  `internal/actors/workflow_handler.go` (Init/Resume/recovery); `internal/repositories/journal.go`;
  `internal/handlers/resolve_awakeable.go`.
- Related: [ADR-0002](0002-ergo-actor-model-for-workflow-execution.md),
  [ADR-0003](0003-in-memory-repositories-by-default.md),
  [ADR-0011](0011-threading-model-and-foreach.md) (threads referenced by journal entries),
  [ADR-0019](0019-object-store-payload-externalization.md).
