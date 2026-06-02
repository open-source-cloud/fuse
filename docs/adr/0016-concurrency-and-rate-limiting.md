# 0016. Multi-scope concurrency control and rate limiting

- Status: Accepted
- Date: 2026-06-01
- Deciders: FUSE maintainers

> Backfill ADR: records a foundational decision already implemented.

## Context and Problem Statement

Workflow functions call real systems (HTTP APIs, the LLM providers in
[ADR-0006](0006-llm-provider-abstraction-and-multi-provider-strategy.md), databases) that have
their own limits. The engine must cap how many executions run at once and how fast they fire —
globally per function, per workflow, and scoped to a key (e.g. per tenant/user) — without
deadlocking the actor pipeline.

## Decision Drivers

- Limit concurrency at multiple scopes: per-function, per-workflow, and per-key.
- Rate-limit calls over a time window, with a choice to queue or reject.
- Don't deadlock the `WorkflowFunc` worker pool or the workflow supervisor.

## Considered Options

- **In-process semaphores (per scope) + token-bucket rate limiters**, config-declared per function/workflow.
- **A single global worker-pool size** as the only throttle.
- **An external rate-limiting service** (e.g. Redis-based).

## Decision Outcome

Chosen option: **in-process semaphores plus token buckets, declared in config.**
`internal/concurrency/Manager` holds three semaphore maps — `functions`, `workflows`, `keyed`
(`"scope:keyValue"`). Limits are declared as `ConcurrencyConfig{Limit (1–1000), Key}`
(`pkg/workflow/concurrency.go`); a non-empty `Key` scopes the semaphore per key value.
Semaphores are FIFO (`internal/concurrency/semaphore.go`): `Acquire` blocks for a slot (used for
function concurrency in `internal/actors/workflow_func.go`), `TryAcquire` is non-blocking (used
to admit workflows without blocking the supervisor — requeue if full).

Rate limiting (`pkg/workflow/ratelimit.go`: `RateLimitConfig{Limit, Period, Key, Strategy}`)
uses per-key token buckets (`internal/concurrency/token_bucket.go`, refill = limit/period). The
**queue** strategy blocks until a token is available; the **reject** strategy returns an error
immediately (surfaced as a `FunctionError` result). `WorkflowFunc` acquires the concurrency slot
(blocking, `defer release()`) and applies the rate limiter before executing the function.

### Consequences

- Good: throttling at the right granularity (function / workflow / key); queue-vs-reject is a
  per-function choice; no external dependency for single-node throughput control.
- Good: blocking vs `TryAcquire` is chosen deliberately — block a pool worker, but never block
  the supervisor (requeue instead) to avoid deadlock.
- Bad: limits are **per process**, not cluster-wide — under HA each node enforces its own caps,
  so global limits are approximate across nodes.
- Neutral: keyed limits parse a key expression at runtime; bad keys degrade to unscoped.

## Pros and Cons of the Options

### In-process semaphores + token buckets (chosen)

- Good: precise multi-scope control, queue/reject, zero external deps, fits the actor pool.
- Bad: per-process only (not globally coordinated in HA).

### Single global worker-pool size

- Good: trivial.
- Bad: one knob can't express per-function/per-key limits or rate windows.

### External rate-limiting service (Redis)

- Good: cluster-wide coordination.
- Bad: heavy dependency, network hop per call, new failure mode; unnecessary for current needs.

## More Information

- Code: `internal/concurrency/` (`manager.go`, `semaphore.go`, `rate_limiter.go`,
  `token_bucket.go`); config `pkg/workflow/concurrency.go`, `pkg/workflow/ratelimit.go`;
  enforcement in `internal/actors/workflow_func.go`; DI `internal/app/di/concurrency.go`.
- Related: [ADR-0002](0002-ergo-actor-model-for-workflow-execution.md),
  [ADR-0018](0018-high-availability-and-clustering.md) (why limits are per-node under HA).
