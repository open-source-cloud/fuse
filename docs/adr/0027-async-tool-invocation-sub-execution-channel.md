# 0027. Async tool invocation via a sub-execution correlation channel

- Status: Proposed
- Date: 2026-06-02
- Deciders: FUSE maintainers

## Context and Problem Statement

Phase B ([ADR-0007](0007-agent-reasoning-loop-and-tools-from-functions.md)) restricts `ai/agent`
tools to **synchronous** functions. The reason is mechanical: asynchronous and intercepted
functions (`system/sleep`, `system/wait`, `system/subworkflow`, `system/foreach`,
`fuse/pkg/logic/timer`) deliver their result via `ExecutionInfo.Finish`, which the internal
transport rebinds to send an `AsyncFunctionResultMessage` to the **WorkflowHandler** — not back to
the agent goroutine that requested the tool. To let agents call async tools (and to enable the
orchestrator mode in [ADR-0026](0026-agent-as-orchestrator-mode.md)), we need a way to correlate
an async tool's result back to the waiting agent.

## Decision Drivers

- Reuse the existing async `Finish` / journal path
  ([ADR-0010](0010-durable-execution-journal-and-replay.md)) rather than inventing a new one.
- Never block the `WorkflowFunc` worker pool; the agent already runs in its own goroutine.
- Respect the ergo constraint that goroutines outside `HandleMessage` use `Node().Send`, not
  `Process.Send`.
- Correlate by execution id; support per-tool timeouts and cancellation.
- Compose with threading/fork-join ([ADR-0011](0011-threading-model-and-foreach.md)).

## Considered Options

- **A — Correlation registry + waiter channel.** The agent allocates a child exec id per async
  tool call and registers a waiter (a channel) in the WorkflowHandler keyed by that id; when the
  async result arrives, the handler resolves the waiter instead of (or in addition to) folding it
  into aggregated output. The agent goroutine selects on the channel with a timeout.
- **B — Tool-as-sub-workflow.** Model each async tool call as a real child sub-execution
  (`system/subworkflow` style) and have the agent await its journaled completion.
- **C — Block on an actor future.** The agent blocks on a future resolved by a dedicated actor
  message round-trip.

## Decision Outcome

**Proposed — leaning to Option A, deferred** until Phase B is exercised in real workflows. A
correlation registry keyed by child exec id, delivering to a per-agent waiter channel with a
timeout, reuses the proven async path with the least new machinery and keeps results routable for
replay. Option B is the natural escalation if tool calls need full sub-workflow semantics.

### Consequences

- Good: lifts the sync-only limitation; unblocks orchestrator mode and richer agents.
- Good: reuses async completion + journal; worker pool stays free.
- Bad: introduces a correlation/lifetime concern (leaked waiters, timeouts, cancellation on agent
  termination) that must be carefully managed.
- Neutral: synchronous tools continue to work unchanged via the inline path.

## Pros and Cons of the Options

### A — Correlation registry + waiter channel (chosen lean)
- Good: minimal, reuses async `Finish`; explicit timeouts.
- Bad: waiter lifecycle/cleanup must be bulletproof.

### B — Tool-as-sub-workflow
- Good: uniform with the engine; full journaling.
- Bad: heavier per tool call; more latency and bookkeeping.

### C — Block on an actor future
- Good: conceptually simple.
- Bad: risks holding resources; awkward with ergo Sleep-state send rules.

## More Information

- Mechanism this lifts: the `Finish` rebinding in `internal/packages/transport/internal.go` and
  the sync-only exclusion predicate in `internal/packages/agent_tools.go`.
- Related: [ADR-0007](0007-agent-reasoning-loop-and-tools-from-functions.md) (the limitation),
  [ADR-0026](0026-agent-as-orchestrator-mode.md) (a primary consumer),
  [ADR-0010](0010-durable-execution-journal-and-replay.md),
  [ADR-0011](0011-threading-model-and-foreach.md).
