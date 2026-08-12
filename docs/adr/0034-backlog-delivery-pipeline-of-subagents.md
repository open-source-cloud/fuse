# 0034. Deliver backlog work through a pipeline of specialised subagents

- Status: Accepted
- Date: 2026-08-11
- Deciders: FUSE maintainers

## Context and Problem Statement

The work queue (`BACKLOG_V2.md`) has a property that ordinary backlogs do not: **its tasks were
derived from a capability report about the code, not from the code.** The document says so in its
own header — "The report describes the code; it is not the code" — and task F-01 exists purely to
verify three findings before anything is built. At least one premise is already suspect: A-03
describes a restart re-invoking a worker "with a new execID", while `replayPendingThread`
(`internal/workflow/workflow.go`) appears to reuse the pending step's existing execID.

An agent that treats such a backlog as ground truth will implement around bugs that do not exist,
and will do it confidently. The failure is silent and expensive.

Two further properties raise the cost of getting this wrong. Tier B makes a **published protocol
spec** the product pillar — once an SDK exists, a wire change breaks strangers, so spec work must
precede and outrank implementation. And F-02 (a real-process crash-resume harness) **blocks all of
Tier A** while gating the agent-execution-model decision, so "correct" is undefined until it exists.

How should engine work be executed so that verification, planning, implementation and review are
distinct, auditable steps rather than one model's single pass?

## Decision Drivers

- A premise that does not reproduce must **close** its task — that outcome needs to be as easy to
  produce as a fix, and as respected.
- Separation of powers: the actor that reproduces a bug should not be the actor that fixes it, and
  neither should be the actor that judges the fix.
- Engine knowledge (18 journal types, 5 replayed; driver parity; pool size 3) is expensive to
  rediscover per session and must be written down once, citation-grounded.
- Protocol changes need a different craft — and a different gate — from Go changes.
- Portability: whatever we add must live under `.agents/` per [ADR-0009](0009-portable-ai-agent-guidance.md).

## Considered Options

- **One general agent plus the existing rules.** Zero new surface; relies on `.agents/rules/`
  already being loaded. But nothing forces verification before implementation, and nothing stops a
  wire-surface change from landing without a spec.
- **A skill per backlog tier.** Knowledge packs only, no role separation — the same context that
  wrote the code also reviews it, which is the review failure mode we are trying to avoid.
- **A pipeline of specialised subagents plus knowledge packs** (chosen).

## Decision Outcome

Chosen option: **seven subagents under `.agents/agents/`, orchestrated by a `backlog-task` skill,
backed by seven engine knowledge packs under `.agents/skills/`.**

The pipeline is: locate & scope → **verify premise** → research → **plan** → implement → **review**
→ gates & land. Each stage is a separate agent with its own tool allowlist and explicit boundaries:

| Agent | Owns | Must not |
| --- | --- | --- |
| `fuse-premise-verifier` | Reproduces a claim with an executed failing test | Fix anything |
| `fuse-engine-researcher` | Read-only maps, every claim CONFIRMED vs INFERRED | Write any file |
| `fuse-implementation-planner` | Files, 3-level test plan, parity, migration, ADR call | Plan an unverified premise |
| `fuse-protocol-spec-author` | B-01 spec + ADR (Markdown only) | Write engine Go |
| `fuse-go-implementer` | Test-first implementation, runs the gates | Bundle tasks; drive-by fixes |
| `fuse-e2e-harness-engineer` | F-02 real-process SIGKILL harness | Fix engine bugs it finds |
| `fuse-code-reviewer` | Adversarial pre-merge review | Edit the code |

Two boundaries are drawn explicitly because they are the ones dispatch gets wrong: a test that
kills and restarts a real OS process belongs to `fuse-e2e-harness-engineer` and involves no
production Go; everything else, including all engine fixes, belongs to `fuse-go-implementer`. And
`fuse-protocol-spec-author` is the *first* call for B-01 or any wire-surface change — ahead of the
planner, which plans Go only against a spec clause that already exists.

`.claude/agents` symlinks into `.agents/agents/`, matching the existing wiring for rules, skills and
commands.

### Consequences

- Good: "the premise did not reproduce" becomes a first-class, cheap outcome instead of an
  awkward one, which is what the backlog's own rules demand.
- Good: engine knowledge is written down once, with file:symbol citations, and each pack names the
  backlog task that owns each known defect — so no agent mistakes a tracked bug for the contract.
- Good: a wire-surface change cannot reach code without passing through a spec author.
- Bad: more surface to keep current. A pack that drifts is worse than no pack, because it is
  trusted. Packs cite symbols as well as line numbers so a stale offset degrades to a re-grep
  rather than a wrong answer.
- Neutral: the pipeline is advisory. A maintainer can invoke any agent directly.

## More Information

- Entry point: `.agents/skills/backlog-task/SKILL.md` (`/backlog-task F-02`), which carries a
  routing table for every backlog id with a premise-confidence column.
- Knowledge packs: `durable-execution-internals`, `crash-resume-testing`,
  `persistence-and-migrations`, `observability-tracing`, `function-package-authoring`,
  `remote-node-protocol`, `capability-registry-and-agents`.
- Layout and symlink rationale: [ADR-0009](0009-portable-ai-agent-guidance.md).
- The durability contract these packs describe: [ADR-0010](0010-durable-execution-journal-and-replay.md).
- Tier D (Enterprise) is out of scope here: it is a separate repo consuming Core's public API.
