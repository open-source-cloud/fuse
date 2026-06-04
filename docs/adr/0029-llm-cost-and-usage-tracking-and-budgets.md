# 0029. LLM cost & token-usage tracking and budgets

- Status: Accepted (Phase A — usage visibility — shipped; budget enforcement deferred)
- Date: 2026-06-02
- Deciders: FUSE maintainers

## Context and Problem Statement

Every LLM call returns a `llm.Usage` (prompt/completion/total tokens), and the `ai/agent` node
aggregates it into its output
([ADR-0007](0007-agent-reasoning-loop-and-tools-from-functions.md)). But there is **no
engine-level** view of usage: no per-workflow / per-tenant / per-execution cost attribution, no
budget or hard limit on spend, and usage is not surfaced into metrics or traces
([ADR-0020](0020-observability-metrics-tracing-execution-traces.md)). An agent loop bounded in
*iterations* can still be unbounded (or surprising) in *cost*, and operators have no way to cap or
even observe spend.

## Decision Drivers

- Cost attribution and control per workflow / tenant / execution.
- Surface usage into observability ([ADR-0020](0020-observability-metrics-tracing-execution-traces.md)).
- Compose with the multi-scope limiter ([ADR-0016](0016-concurrency-and-rate-limiting.md)) and the
  (deferred) tenancy/secrets model ([ADR-0008](0008-settings-environments-and-secrets-management.md)).
- Prevent runaway spend with a deterministic terminal outcome (like the existing `maxIterations`
  cap).

## Considered Options

- **A — Metrics only.** Emit usage as metrics and trace attributes; no enforcement.
- **B — Budget enforcement.** A per-scope token/cost budget checked in the agent/provider seam;
  exceeding it ends the run with a terminal error (mirroring `maxIterations`).
- **C — External metering / billing integration.** Stream usage to an external metering system.

## Decision Outcome

**Proposed — phased, deferred.** Start with **Option A** (instrument usage through the existing
observability stack so cost is *visible*), then add **Option B** (budgets) composing with the
multi-scope limiter from [ADR-0016](0016-concurrency-and-rate-limiting.md) and any future tenancy
scope. **Option C** is an optional backend once B exists. The cost model (per-provider/per-model
pricing) is a prerequisite and is left to the follow-up.

### Consequences

- Good: makes spend observable immediately; enforcement arrives without re-instrumenting.
- Good: reuses the limiter's scope model rather than inventing a parallel one.
- Bad: accurate cost requires a maintained per-model pricing table.
- Neutral: enforcement (B) is deferred behind visibility (A).

## Pros and Cons of the Options

### A — Metrics only (chosen first)
- Good: cheap, immediately useful, no behavioural change.
- Bad: no protection against runaway spend.

### B — Budget enforcement
- Good: hard cost control; deterministic terminal outcome.
- Bad: needs a pricing model and scope wiring.

### C — External metering
- Good: enterprise billing/accounting.
- Bad: external dependency; overkill until A/B exist.

## More Information

- **Phase A (usage visibility) shipped**: `ai/chat` and `ai/agent` emit per-call token usage as
  Prometheus counters `fuse_llm_tokens_total{function,provider,model,type}` (type ∈ prompt|
  completion) and `fuse_llm_calls_total{function,provider,model,status}`
  (`internal/metrics/registry.go`). A narrow `ai.UsageRecorder` port
  (`internal/packages/functions/ai/usage.go`) keeps the ai package free of the prometheus
  dependency; a metrics-backed adapter is injected via `packages.NewInternal`
  (`internal/packages/usage_recorder.go`). Usage is still also returned in each node's `usage`
  output. **Deferred**: Option B budget enforcement (needs a per-provider/per-model pricing table)
  and Option C external metering.
- Current state: `pkg/llm/provider.go` (`Usage`), `internal/packages/functions/ai/agent.go`
  (per-run aggregation), `internal/packages/functions/ai/chat.go`.
- Related: [ADR-0006](0006-llm-provider-abstraction-and-multi-provider-strategy.md),
  [ADR-0007](0007-agent-reasoning-loop-and-tools-from-functions.md),
  [ADR-0016](0016-concurrency-and-rate-limiting.md),
  [ADR-0020](0020-observability-metrics-tracing-execution-traces.md).
