# 0005. AI agents as workflow nodes; phased roadmap

- Status: Accepted
- Date: 2026-06-01
- Deciders: FUSE maintainers

## Context and Problem Statement

We want FUSE to support n8n-style **agent workflows**: a trigger starts a flow, an
**AI agent** is given tools and reasons in a loop to accomplish a task, and the
result flows on to downstream nodes. The question is *how* an "agent" fits the
engine: is it a node inside the existing deterministic graph, or a new
orchestration mode where the LLM drives the whole flow? And how much do we build
at once?

## Decision Drivers

- Fit the existing graph / journal / replay / supervision model (see
  [ADR-0002](0002-ergo-actor-model-for-workflow-execution.md)).
- Ship usable value incrementally; avoid a big-bang rewrite.
- Reuse what already exists: triggers ([ADR-0004](0004-multi-trigger-workflow-initiation.md)),
  the package/function registry (tools), and async execution.
- Keep the door open to a more autonomous "agent owns the flow" mode later.

## Considered Options

- **Agent-as-node** — an `ai/agent` node inside a normal workflow graph; it owns an
  internal reasoning/tool-calling loop and emits a result like any node.
- **Agent-as-orchestrator** — the LLM dynamically decides which nodes/tools run
  next; no fixed graph.
- **Both, phased** — agent-as-node first, orchestrator later on the same foundations.

## Decision Outcome

Chosen option: **both, phased — starting with agent-as-node.** An agent is an
ordinary package function with `transport.Internal`, so it slots into the graph
with **zero executor changes**; its inputs/outputs use the existing parameter-schema
metadata and downstream edges read its output normally. The autonomous
orchestrator mode is deferred until the provider and tool-loop foundations exist.

Roadmap:

- **Phase A — provider layer + `ai/chat`** (Accepted, shipped): the LLM provider
  abstraction ([ADR-0006](0006-llm-provider-abstraction-and-multi-provider-strategy.md))
  and a no-tools chat node. Delivered in PR #69.
- **Phase B — `ai/agent` tool-calling loop**
  ([ADR-0007](0007-agent-reasoning-loop-and-tools-from-functions.md)): the agent uses
  existing FUSE functions as tools.
- **Phase C — breadth**: native Anthropic provider, streaming, and async-tool support.
- **Later — agent-as-orchestrator**: built on the Phase B/C foundations.

### Consequences

- Good: fast, low-risk path to a working agent; deterministic and observable via the
  existing graph/journal.
- Good: triggers, tools, async, and supervision are reused, not rebuilt.
- Neutral: the autonomous mode is explicitly deferred (a future ADR will cover it).
- Bad: agent-as-node constrains an agent to a single graph position; cross-node
  autonomy waits for the orchestrator phase.

## Pros and Cons of the Options

### Agent-as-node

- Good: zero executor change; deterministic; fits journal/replay; ships fast.
- Bad: the agent is one step, not the whole flow.

### Agent-as-orchestrator

- Good: maximally flexible/autonomous.
- Bad: large departure from the deterministic model; harder to make fault-tolerant
  and replayable; much bigger first step.

### Both, phased

- Good: usable now, ambitious later, on shared foundations.
- Bad: requires sequencing discipline across phases.

## More Information

- Phase A: PR #69; code in `internal/packages/functions/ai/` (`ai/chat`).
- Related: [ADR-0006](0006-llm-provider-abstraction-and-multi-provider-strategy.md),
  [ADR-0007](0007-agent-reasoning-loop-and-tools-from-functions.md),
  [ADR-0008](0008-settings-environments-and-secrets-management.md) (per-context API keys
  for agents).
