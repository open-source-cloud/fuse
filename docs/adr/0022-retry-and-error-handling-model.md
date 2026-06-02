# 0022. Retry and error-handling model

- Status: Accepted
- Date: 2026-06-01
- Deciders: FUSE maintainers

> Backfill ADR: records a foundational decision already implemented.

## Context and Problem Statement

Workflow nodes call fallible systems (HTTP APIs, LLM providers, databases). When a node fails,
the engine must decide what happens next: retry it, route the workflow down an alternative path,
or fail the workflow. The durable journal ([ADR-0010](0010-durable-execution-journal-and-replay.md))
records *that* a node failed and was retried, but the *policy* — how many times, with what delay,
and what happens when retries run out — is a separate execution-semantics decision that needs to
be explicit and configurable per node.

## Decision Drivers

- Transient failures (timeouts, 5xx, rate limits) should be retried automatically with backoff.
- Authors need per-node control (a flaky HTTP call ≠ a deterministic transform).
- A graph-level way to handle "this step failed" without crashing the whole workflow.
- Deterministic, journaled, and replay-safe ([ADR-0010](0010-durable-execution-journal-and-replay.md)).

## Considered Options

- **Per-node retry policy with backoff + first-class error edges, with an ordered fallback
  retry → error-edge → fail-workflow.**
- **Retry-only** (retry with backoff; on exhaustion, always fail the workflow).
- **Fail-fast** (no retries; any node error fails the workflow).
- **A single global retry setting** for all nodes.

## Decision Outcome

Chosen option: **per-node `RetryPolicy` + error edges, evaluated as an ordered fallback.**

- `RetryPolicy` (`internal/workflow/retry.go`) on `NodeSchema.Retry`: `MaxAttempts` (0–100, 0 = no
  retries) and a `BackoffConfig{Type, InitialInterval, MaxInterval, Multiplier}` where
  `BackoffType` is `fixed | exponential | linear`. `DefaultRetryPolicy` = 3 attempts, exponential,
  1s → 30s cap, ×2; `DelayFor(attempt)` computes the per-attempt delay.
- A `RetryTracker` (`retry_tracker.go`) counts attempts per `execID`.
- `Workflow.HandleNodeFailure` (`workflow.go`) is the decision point: if a retry policy applies and
  `attempts <= MaxAttempts`, it returns a `RetryFunctionAction` with the computed delay and journals
  `step:retrying`; once retries are exhausted it clears the tracker and looks for **error edges**
  (`EdgeSchema.OnError`) — if present, execution continues down the first error edge (marking the
  source thread finished when the edge crosses threads, [ADR-0011](0011-threading-model-and-foreach.md));
  if none, it returns nil and the caller transitions the workflow to `StateError`. A failed node can
  also be re-run on demand via the manual `RetryNode` path.

Logical failures are carried in the function's `FunctionOutput` (status `error`), distinct from Go
errors for unexpected execution faults, so the engine always gets a result to act on.

### Consequences

- Good: transient failures self-heal with tuned backoff; authors control behavior per node;
  error edges express "on failure, do X" as ordinary graph routing; everything is journaled and
  replay-safe.
- Good: composes with timeouts ([ADR-0023](0023-timeout-enforcement-model.md)) — a timeout fires a
  failure that flows through this same path.
- Bad: retrying a non-idempotent node can duplicate side effects (the engine retries the function,
  it can't know if the external effect already happened) — authors must design idempotent steps or
  use idempotency keys ([ADR-0017](0017-idempotency-check-and-set.md)).
- Neutral: only the first error edge is followed; multiple error branches aren't fanned out.

## Pros and Cons of the Options

### Per-node policy + error edges (chosen)

- Good: granular, expressive (retry *and* alternative routing), journaled, sensible default.
- Bad: duplicate-side-effect risk on retry; per-node config to author.

### Retry-only

- Good: simpler — one mechanism.
- Bad: no graph-level recovery; a permanently failing step always kills the workflow.

### Fail-fast

- Good: trivial; no duplicate-effect risk.
- Bad: no resilience to transient faults — unacceptable for real integrations.

### Single global retry setting

- Good: nothing to configure per node.
- Bad: one policy can't fit both flaky I/O and deterministic compute; no error-routing.

## More Information

- Code: `internal/workflow/retry.go` (`RetryPolicy`, `BackoffConfig`, `DelayFor`),
  `retry_tracker.go`, `workflow.go` (`HandleNodeFailure`, `findErrorEdges`, `RetryNode`),
  `node_schema.go` (`Retry`), `edge_schema.go` (`OnError`);
  retry/error actions handled in `internal/actors/workflow_handler.go`.
- Related: [ADR-0010](0010-durable-execution-journal-and-replay.md),
  [ADR-0023](0023-timeout-enforcement-model.md),
  [ADR-0017](0017-idempotency-check-and-set.md),
  [ADR-0013](0013-workflow-schema-model-and-input-mapping.md).
