# 0026. Agent-as-orchestrator mode

- Status: Proposed
- Date: 2026-06-02
- Deciders: FUSE maintainers

## Context and Problem Statement

[ADR-0005](0005-ai-agents-as-workflow-nodes-phased-roadmap.md) chose **agent-as-node** first and
explicitly deferred the autonomous **agent-as-orchestrator** mode, where "the LLM dynamically
decides which nodes/tools run next; no fixed graph." With Phase B
([ADR-0007](0007-agent-reasoning-loop-and-tools-from-functions.md)) now shipping the in-node
tool-calling loop, the foundations exist to design that mode. The open question: how does an
LLM-driven, dynamic flow fit FUSE's deterministic graph / journal / replay / supervision model
without sacrificing replayability and fault tolerance?

## Decision Drivers

- Preserve durable replay determinism ([ADR-0010](0010-durable-execution-journal-and-replay.md)):
  every non-deterministic LLM decision must be journaled so a replay is faithful.
- Reuse the tool loop ([ADR-0007](0007-agent-reasoning-loop-and-tools-from-functions.md)), the
  provider layer ([ADR-0006](0006-llm-provider-abstraction-and-multi-provider-strategy.md)), and
  async completion rather than a second orchestration engine.
- Fit supervision/HA ([ADR-0018](0018-high-availability-and-clustering.md)) and threading
  ([ADR-0011](0011-threading-model-and-foreach.md)).
- Keep authorability, observability ([ADR-0020](0020-observability-metrics-tracing-execution-traces.md)),
  and safety (bounded autonomy, cost limits — see
  [ADR-0029](0029-llm-cost-and-usage-tracking-and-budgets.md)).

## Considered Options

- **A — Stay agent-as-node only.** No orchestrator; compose autonomy by chaining agent nodes.
- **B — Bounded orchestrator node.** A special node that owns a *sub-graph* and dynamically routes
  to/spawns sub-executions via the async sub-execution channel
  ([ADR-0027](0027-async-tool-invocation-sub-execution-channel.md)); the graph topology is still
  declared, but traversal is LLM-driven within it.
- **C — Full dynamic planner.** The LLM rewrites/extends the graph at runtime with no
  pre-declared topology.

## Decision Outcome

**Proposed — leaning to Option B, deferred.** A bounded orchestrator node keeps the engine's
determinism (each routing decision is journaled as an event; replay re-applies the recorded
decisions instead of re-querying the model) while delivering meaningful autonomy. It depends on
[ADR-0027](0027-async-tool-invocation-sub-execution-channel.md) (so the orchestrator can await
async sub-steps) and a context/memory model
([ADR-0028](0028-agent-prompt-context-and-memory-model.md)). The full dynamic planner (C) is out
of scope until B is proven. No work starts until those prerequisites land and Phase B usage
informs the routing contract.

### Consequences

- Good: a credible path to autonomy that stays replayable and supervised.
- Good: reuses tools, providers, async, and the journal rather than a parallel runtime.
- Bad: journaling LLM decisions and bounding autonomy is non-trivial; needs careful safety limits.
- Neutral: the "no fixed graph" ambition (C) is intentionally deferred.

## Pros and Cons of the Options

### A — Agent-as-node only
- Good: simplest; already shipped; fully deterministic.
- Bad: no cross-node autonomy; authors wire all branches.

### B — Bounded orchestrator node (chosen lean)
- Good: autonomy within a declared, replayable boundary; reuses existing machinery.
- Bad: requires the async sub-execution channel and a decision-journaling design.

### C — Full dynamic planner
- Good: maximal flexibility.
- Bad: hard to make replayable/fault-tolerant; large departure from the engine model.

## More Information

- Related: [ADR-0005](0005-ai-agents-as-workflow-nodes-phased-roadmap.md) (deferred this mode),
  [ADR-0007](0007-agent-reasoning-loop-and-tools-from-functions.md) (the tool loop it builds on),
  [ADR-0027](0027-async-tool-invocation-sub-execution-channel.md),
  [ADR-0010](0010-durable-execution-journal-and-replay.md),
  [ADR-0028](0028-agent-prompt-context-and-memory-model.md).
