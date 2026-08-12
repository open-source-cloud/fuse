# Backlog task dossiers

One working file per task in [`BACKLOG_V2.md`](../../BACKLOG_V2.md), plus the three unfiled findings
raised during F-01 verification. 45 documents.

**`BACKLOG_V2.md` remains the source of truth for task text** — tier, size, dependencies, the dispatch
order, the sprints, the HARD GATE and the rules for the implementing agent. These dossiers never
replace it. Each one quotes its entry verbatim under *What the backlog says* and then adds what the
backlog does not carry: whether the premise has actually been executed, the real code area with
`file:line`, which agent runs first, a derived *Done when* checklist, an explicit *Out of scope*, the
ADR/CHANGELOG obligations, and a dated *Log*.

They are **working files**. Edit them as a task moves — flip `Status`, tick a `Done when` box, append
to `Log`. A dossier that still says "Not started" after the PR merged is a stale dossier, not a record.

## Starting a task

```
/backlog-task <ID>          # e.g. /backlog-task A-07
/backlog-task               # proposes the next task and stops for confirmation
```

The skill runs the fixed pipeline — **locate → verify → research → plan → implement → review → land** —
delegating each stage to a `fuse-*` agent. Read the dossier first; hand its *Route* section to the
agent named in *First agent*. Full contract: [`.agents/skills/backlog-task/SKILL.md`](../../.agents/skills/backlog-task/SKILL.md),
agents in [`.agents/agents/`](../../.agents/agents/).

Two rules the dossiers assume rather than repeat:

- **Verify before fixing.** Every task cites a report, not the code. `NOT-REPRODUCED` ends the
  pipeline and closes the task — that is a success, not a failure.
- **A test reproducing the bug lands before the fix**, in the same PR, failing in the first commit.

### Reading the columns

| Column | Values |
| --- | --- |
| **status** | `Closed` · `Premise verified` (stage 1 done, implementation open) · `Not started` · `Blocked` (a declared blocker is open) |
| **premise** | `REPRODUCED`/`EXECUTED` — a test was run · `READ` — control flow read, not executed · `VERIFY` — report-derived, stage 1 must run · `VERIFY ⚠` — the entry's wording diverges from the code · `net-new` — nothing to reproduce, it is construction |
| **first agent** | Stage the task starts at. `†` marks a dossier whose *Route* section deviates from the default pipeline — read it before delegating. |

Sizes are the backlog's: **S** ≈ under a day, **M** ≈ a few days, **L** ≈ a week or more with design
risk. The backlog sizes no Tier D item.

---

## Tier F — Foundation

*Verification and the safety net. Nothing else starts, and no execution-model decision gets made,
until these pass.*

| id | title | size | status | premise | first agent |
| --- | --- | --- | --- | --- | --- |
| [F-01](F-01.md) | Verify the three high-leverage findings | S | Closed | REPRODUCED ×3 | `fuse-premise-verifier` † |
| [F-02](F-02.md) | Crash-resume e2e harness | M | Not started | net-new (harness) | `fuse-e2e-harness-engineer` † |
| [F-03](F-03.md) | Config validation for driver combinations | S | Not started | VERIFY | `fuse-premise-verifier` |

## Tier A — Core correctness

*The engine's durability claims have to be true before anything is sold on them.*

| id | title | size | status | premise | first agent |
| --- | --- | --- | --- | --- | --- |
| [A-01](A-01.md) | Input payload on workflow trigger | S | Not started | VERIFY | `fuse-premise-verifier` |
| [A-02](A-02.md) | Replay the intercepted system functions | M | Blocked | REPRODUCED (subworkflow) · VERIFY ⚠ (awakeable, sleep) | `fuse-premise-verifier` † |
| [A-03](A-03.md) | Replay semantics for pending remote steps | M | Blocked | VERIFY ⚠ | `fuse-premise-verifier` |
| [A-04](A-04.md) | Trace context end to end | M | Premise verified | REPRODUCED | `fuse-implementation-planner` |
| [A-05](A-05.md) | Pin runs to a schema version | M | Not started | VERIFY | `fuse-premise-verifier` |
| [A-06](A-06.md) | Stop the shared-pointer live edit | S | Premise verified | REPRODUCED | `fuse-implementation-planner` |
| [A-07](A-07.md) | Fix duplicate journal append on replay | S | Not started | VERIFY ⚠ | `fuse-premise-verifier` |
| [A-08](A-08.md) | Distinguish retry attempts | S | Not started | VERIFY | `fuse-premise-verifier` |
| [A-09](A-09.md) | Payload write amplification | M | Not started | VERIFY | `fuse-premise-verifier` |
| [A-10](A-10.md) | Configurable pool size and non-blocking remote calls | S–M | Not started | VERIFY | `fuse-premise-verifier` |
| [A-11](A-11.md) | Address node output by execution | L | Not started | VERIFY | `fuse-premise-verifier` † |

**A-07 is marked "do first"** — it compounds on every restart and corrupts the trace, so every day it
stays open makes every other restart-shaped task noisier to measure. **A-11 needs an ADR before code**
and is the one item in the document that is expensive to reverse.

**Unresolved, and it decides how this tier is sequenced:** F-02's heading says *"blocks all of A"*,
but the Dispatch order graph routes only A-02, A-03 and C-12 through F-02, and Sprint 1 lists A-06 and
A-07 alongside it. Both readings are in the backlog. Every Tier A dossier records the disagreement and
points at [F-02](F-02.md); none of them picks a side. **A maintainer call.**

## Tier B — Extension protocol and SDKs

*The pillar. **Everything in this tier is net-new** — the engine declares `HTTP` and `gRPC` in
`pkg/transport/type.go` and implements neither. Nothing here describes a capability FUSE has today.*

| id | title | size | status | premise | first agent |
| --- | --- | --- | --- | --- | --- |
| [B-01](B-01.md) | Specify the remote node protocol | M | Not started | net-new | `fuse-protocol-spec-author` |
| [B-02](B-02.md) | Implement the remote transport in Core | M | Blocked | net-new | `fuse-implementation-planner` † |
| [B-03](B-03.md) | Capability registry | M | Blocked | net-new | `fuse-premise-verifier` † |
| [B-04](B-04.md) | MCP ingestion | M | Blocked | net-new | `fuse-implementation-planner` |
| [B-05](B-05.md) | SDK conformance suite | M | Blocked | net-new | `fuse-protocol-spec-author` † |
| [B-06](B-06.md) | SDK: TypeScript / Node | M | Blocked | net-new | `fuse-protocol-spec-author` † |
| [B-07](B-07.md) | SDK: PHP / Laravel | M | Blocked | net-new | `fuse-protocol-spec-author` † |
| [B-08](B-08.md) | Worker registration lifecycle | S–M | Blocked | net-new | `fuse-protocol-spec-author` |
| [B-09](B-09.md) | Guard sub-workflow recursion | S | Not started | VERIFY | `fuse-premise-verifier` |
| [B-10](B-10.md) | Pass input into sub-workflows | S | Blocked | VERIFY ⚠ | `fuse-premise-verifier` |

**B-01 blocks every other B task — do it before writing any Go.** Protocol tasks invert the default
pipeline: the spec comes first. B-06 and B-07 are built **outside this repo**; no pipeline agent writes
TypeScript or PHP, and an engine gap found while building an SDK is *filed as a Core task*, not patched
from the SDK side. B-09 and B-10 are ordinary engine tasks that happen to sit in this tier.

## Tier C — Agents in Core

*What makes it an agent engine rather than a workflow engine. `ai/agent` and `ai/chat` exist as async
node functions; **every capability below is a task, not a feature**. Where a premise reads `VERIFY` or
`READ`, what is confirmed is the **gap** — a symbol with no caller, a route that is absent — never the
proposed replacement.*

| id | title | size | status | premise | first agent |
| --- | --- | --- | --- | --- | --- |
| [C-01](C-01.md) | Agent node ↔ registry tool binding | M | Blocked | VERIFY | `fuse-premise-verifier` |
| [C-02](C-02.md) | Memory as a first-class concept | M | Not started | net-new | `fuse-engine-researcher` † |
| [C-03](C-03.md) | Model gateway | M | Not started | net-new | `fuse-engine-researcher` † |
| [C-04](C-04.md) | GenAI telemetry and cost | M | Blocked | VERIFY | `fuse-premise-verifier` |
| [C-05](C-05.md) | Human gate payload schema | M | Blocked | net-new | `fuse-engine-researcher` † |
| [C-06](C-06.md) | Durable timers and expiry sweeper | M | Not started | VERIFY | `fuse-premise-verifier` |
| [C-07](C-07.md) | Expose awakeable IDs | S | Not started | VERIFY | `fuse-premise-verifier` |
| [C-08](C-08.md) | ForEach state survives restart, and stops silently lying | M | Not started | VERIFY | `fuse-premise-verifier` † |
| [C-09](C-09.md) | Mid-flight run read model | M | Not started | VERIFY | `fuse-premise-verifier` |
| [C-10](C-10.md) | Per-node idempotency convention | M | Blocked | VERIFY | `fuse-protocol-spec-author` † |
| [C-11](C-11.md) | Claim failure must fail closed | S | Not started | VERIFY | `fuse-premise-verifier` |
| [C-12](C-12.md) | Multi-thread resume correctness | M | Blocked | VERIFY ⚠ | `fuse-premise-verifier` |

**C-08 has a blocking prerequisite the backlog entry does not mention** — a Postgres enum that has no
`foreach:*` values. Read its first section before anything else; it changes step one of the task.
**C-02 overlaps A-11(a)** — the backlog says do them together, which is design coordination, not
permission to land both in one PR. **C-07 and C-11 are the two cheapest** items in the tier.

## Tier D — Enterprise layer

*Separate repo, separate release cadence, consumes Core's public API only. **Starts once Tier A is
green.** The backlog sizes no Tier D item, and **no `fuse-*` agent runs here** — they are Core agents.
Anything D needs from Core becomes a new Core task the maintainer files and numbers; never an
Enterprise-side change to `core/`, and never a fork.*

| id | title | size | status | premise | first agent |
| --- | --- | --- | --- | --- | --- |
| [D-01](D-01.md) | Tenancy, RBAC, SSO | — | Blocked | net-new | none in this repo |
| [D-02](D-02.md) | Console | — | Blocked | net-new | none in this repo |
| [D-03](D-03.md) | Collaboration | — | Blocked | net-new | none in this repo |
| [D-04](D-04.md) | Cost and usage dashboards | — | Blocked | net-new | none in this repo |
| [D-05](D-05.md) | Audit export | — | Blocked | net-new | none in this repo |
| [D-06](D-06.md) | Packaging | — | Blocked | net-new | none in this repo |

> **Core stays usable without Enterprise.** If a Core feature only makes sense with the commercial
> layer, it belongs in the commercial layer.

## Found during verification — unfiled

*Raised by F-01 verification (2026-08-11), per "do not fix bugs found along the way".* **Not yet
assigned ids or tiers — that is the maintainer's call.** The `V-x` labels are dossier filenames, not
backlog ids; each dossier carries a `Done when` box for assigning a real one.

| id | title | size | status | premise | first agent |
| --- | --- | --- | --- | --- | --- |
| [V-a](V-a.md) | Agent tool calls are invisible to OTel *and* Prometheus | S | Not started | READ | `fuse-premise-verifier` |
| [V-b](V-b.md) | Resume appends a second `subworkflow:started` for one exec id | S | Not started | EXECUTED | `fuse-implementation-planner` † |
| [V-c](V-c.md) | `Workflow.Next` nil-derefs on a missing node, crashing the run | S | Not started | EXECUTED | `fuse-implementation-planner` † |

Sizes here are the dossiers' own reading, not the backlog's — it assigns none. V-a is standing evidence
against B-03's parity requirement; V-b is a constraint inside A-02 rather than a separate PR; V-c is
independent of A-05 and A-06, which each remove *a* way to reach it without making `Next` safe.

---

## Dispatch order

Quoted from `BACKLOG_V2.md`:

```
F-01 ─┬─ F-02 ──┬── A-02, A-03, C-12     (need the harness to define "correct")
      └─ A-06   │
F-03 ────────────┤

A-07 ── independent, do first (compounds every restart)
A-01 ── independent          → B-10
A-04 ────────────────────────→ B-02
A-05, A-08, A-09, A-10 ── independent
A-11 ── ADR before code, overlaps C-02

B-01 ──┬── B-02 ──┬── B-06, B-07
       ├── B-03 ──┴── B-04, C-01
       ├── B-05 (before the first SDK)
       ├── B-08
       └── C-10

D-* ── after Tier A is green
```

## Sprints

Quoted from `BACKLOG_V2.md`:

> **Sprint 1 — prove the engine.** F-01, F-02, F-03, A-07, A-06. Nothing user-visible ships; everything
> after becomes measurable.
>
> **Sprint 2 — write the spec.** B-01, plus A-01, A-04, A-05 in parallel. The spec is a document and a
> decision, not code, and it gates the entire B tier — do not let it slip behind implementation.
>
> **Sprint 3 — the pillar.** B-02, B-05, then B-06. One SDK, one conformance suite, one example that
> survives a restart mid-call.

## The HARD GATE

Quoted from `BACKLOG_V2.md`:

> **Gate — do not choose an agent execution model before F-02 is green.** The report says cycles are
> genuinely supported, sub-workflow spawn works, and awakeables persist. Two of those three already
> carry known restart defects. Choosing on the strength of prose is how you end up with an elegant
> model resting on a primitive that has never been executed under SIGKILL.

F-02 is not started. If a task or a conversation drifts toward that decision, **stop and say so** —
including work in Tier C, whose whole subject is what agents do inside the engine.
