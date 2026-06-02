# 0028. Agent prompt / context & conversation-memory model

- Status: Proposed
- Date: 2026-06-02
- Deciders: FUSE maintainers

## Context and Problem Statement

The Phase B agent ([ADR-0007](0007-agent-reasoning-loop-and-tools-from-functions.md)) assembles its
message list per execution (an optional system prompt, the user task, and the accumulating
assistant/tool turns) and discards it when the run ends. There is **no** conversation memory
across executions, **no** context-window management (a long tool transcript can exceed the model's
limit), and **no** prompt templating or versioning. As agents grow more capable — multi-turn
sessions, long tool transcripts, large tool outputs — we need a deliberate model for how context
is assembled, bounded, and (optionally) persisted.

## Decision Drivers

- Respect model context-window limits; degrade gracefully on overflow.
- Control cost ([ADR-0029](0029-llm-cost-and-usage-tracking-and-budgets.md)) — context size drives
  token spend.
- Preserve replay determinism ([ADR-0010](0010-durable-execution-journal-and-replay.md)): context
  assembly must be reproducible.
- Reuse payload externalization ([ADR-0019](0019-object-store-payload-externalization.md)) for
  large transcripts/tool outputs.
- Compose with redaction/secrets ([ADR-0008](0008-settings-environments-and-secrets-management.md))
  so secrets never enter the prompt/journal.

## Considered Options

- **A — Stateless per-run + context-assembly policy.** Keep per-execution memory (status quo) but
  add an explicit policy: a token budget, drop-oldest / summarize-oldest tool turns, and a
  templated system prompt.
- **B — Persisted conversation/session store.** A pluggable store keyed by a conversation id gives
  cross-run memory (multi-turn agents resume prior context).
- **C — External memory / vector store (RAG).** Retrieve relevant context from an embedding store
  at each step.

## Decision Outcome

**Proposed — phased, deferred.** Near term: **Option A** (a bounded context-assembly policy with a
token budget and a summarization hook) — it directly addresses overflow and cost while keeping runs
reproducible. **Option B** (a session store) is adopted when multi-turn/conversational agents
arrive. **Option C** (RAG) is out of scope here and would be its own ADR. No work starts until
multi-turn requirements are concrete.

### Consequences

- Good: bounds context growth and cost without changing the node contract.
- Good: a clear seam (the assembly policy) that B and C can extend later.
- Bad: summarization introduces its own model calls and non-determinism to journal.
- Neutral: cross-run memory is explicitly deferred to B.

## Pros and Cons of the Options

### A — Stateless + assembly policy (chosen near-term)
- Good: simple, reproducible, addresses overflow/cost now.
- Bad: no memory beyond a single run.

### B — Persisted session store
- Good: true multi-turn memory; familiar UX.
- Bad: new storage, scoping/tenancy, and retention concerns.

### C — External vector/RAG store
- Good: scalable long-term knowledge.
- Bad: heavy dependency; relevance/eval complexity; separate concern.

## More Information

- Current assembly: `internal/packages/functions/ai/agent.go` (per-run message slice).
- Related: [ADR-0007](0007-agent-reasoning-loop-and-tools-from-functions.md),
  [ADR-0019](0019-object-store-payload-externalization.md),
  [ADR-0029](0029-llm-cost-and-usage-tracking-and-budgets.md),
  [ADR-0008](0008-settings-environments-and-secrets-management.md).
