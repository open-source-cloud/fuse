# 0023. Timeout enforcement via actor timers

- Status: Accepted
- Date: 2026-06-01
- Deciders: FUSE maintainers

> Backfill ADR: records a foundational decision already implemented.

## Context and Problem Statement

A node's function can hang (a stuck HTTP call, a slow LLM, an external async step that never
resolves), and a whole workflow can run longer than is acceptable. The engine needs to bound both
without blocking its actor pipeline — execution is message-driven and often async
([ADR-0002](0002-ergo-actor-model-for-workflow-execution.md),
[ADR-0010](0010-durable-execution-journal-and-replay.md)), so a node may be "in flight" with no
goroutine to cancel.

## Decision Drivers

- Bound individual node execution and total workflow duration.
- Must work for async/long-running steps where no blocking call is in progress.
- Don't tie up `WorkflowFunc` pool workers or the handler waiting on timers.
- Cancellable cleanly when the step finishes in time.

## Considered Options

- **Asynchronous ergo `SendAfter` timer messages**, tracked and cancelled per execution.
- **`context.WithTimeout` per execution goroutine** (block on ctx in the function call).
- **No engine-level timeouts** — rely on each function's own client timeouts.

## Decision Outcome

Chosen option: **async timer messages via ergo `SendAfter`.** Two scopes
(`internal/workflow/timeout.go`): per-node `TimeoutConfig.Execution` and per-workflow
`GraphTimeoutConfig.Total` (zero = no timeout). An `ExecutionTimer`
(`internal/actors/execution_timer.go`) starts a node timeout with
`process.SendAfter(target, NewTimeoutMessage(execID), timeout)`, stores the returned
`gen.CancelFunc` keyed by `execID`, and `Cancel(execID)` cancels it when the result arrives
(`CancelAll` on teardown). The workflow-total timeout is a separate `SendAfter` set by the
`WorkflowHandler` at start. When a timer fires, a `TimeoutMessage` reaches the handler, which
fails that execution — and the failure then flows through the retry/error path
([ADR-0022](0022-retry-and-error-handling-model.md)).

Because timeouts are messages, they don't occupy a worker or block the handler, and they work
even when the node is parked waiting on an external async result.

### Consequences

- Good: bounds both node and workflow duration; fits the non-blocking, async, actor-based pipeline;
  covers async/awaitable steps that have no goroutine to cancel.
- Good: a fired timeout reuses the existing failure semantics (retry → error-edge → fail) rather
  than a separate code path.
- Bad: requires explicit cancel bookkeeping (cancel on completion) — a missed cancel would fire a
  spurious timeout (mitigated by `Cancel`/`CancelAll`).
- Neutral: the timer fires the *engine's* notion of timeout; an in-flight external call may keep
  running on the remote side (no forced cancellation of the remote work).

## Pros and Cons of the Options

### Async `SendAfter` timers (chosen)

- Good: non-blocking, works for async steps, cancellable, reuses the failure path.
- Bad: manual cancel tracking; doesn't abort remote work.

### `context.WithTimeout` per goroutine

- Good: idiomatic Go; cancels the in-progress call.
- Bad: assumes a blocking call to cancel — doesn't fit parked/async executions or the message-driven
  model; ties a goroutine to each pending node.

### No engine-level timeouts

- Good: nothing to build.
- Bad: a hung node or runaway workflow has no backstop; relies on every function getting its own
  client timeout right.

## More Information

- Code: `internal/workflow/timeout.go` (`TimeoutConfig`, `GraphTimeoutConfig`);
  `internal/actors/execution_timer.go` (`Start`/`Cancel`/`CancelAll`);
  `internal/actors/workflow_handler.go` (workflow-total timer, `TimeoutMessage` handling);
  `internal/messaging` (`NewTimeoutMessage`).
- Related: [ADR-0002](0002-ergo-actor-model-for-workflow-execution.md),
  [ADR-0022](0022-retry-and-error-handling-model.md),
  [ADR-0010](0010-durable-execution-journal-and-replay.md).
