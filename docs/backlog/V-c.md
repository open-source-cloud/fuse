# V-c · `Workflow.Next` nil-derefs on a missing node, crashing the run

|  |  |
| --- | --- |
| **Tier** | Unfiled (found during verification) — **the maintainer must assign it a tier and an id** |
| **Size** | S — the finding says "cheap to fix independently" |
| **Status** | Not started |
| **Premise** | EXECUTED (`TestWorkflow_Next_AfterLiveSchemaEditDropsInFlightNode_MustNotPanic`) |
| **Depends on** | nothing |
| **Blocks** | nothing. **A-05** and **A-06** each remove *a* way to reach it; neither makes `Next` safe |
| **First agent** | `fuse-implementation-planner` (premise already executed — no second verification pass needed) |
| **Skill packs** | `durable-execution-internals`, `actor-model-patterns` |

## What the backlog says

> `internal/workflow/workflow.go:312` discards the error from `graph.FindNode` (`currentNode, _ := ...`)
> and `:314` immediately dereferences the result, panicking in `internal/workflow/node.go:68` with
> `runtime error: invalid memory address or nil pointer dereference`.
>
> Any missing-node condition — a live schema edit, replay against a changed schema, a bad audit entry —
> takes down the workflow actor instead of failing the run. A-05 (version pinning) and A-06 (deep copy)
> each remove *a* way to reach it; neither makes `Next` safe. Cheap to fix independently.

## Premise status

**EXECUTED**, during F-01.3, under real Go 1.26 in a `golang:1.26` container (there is no host Go). The
reproducing test is `TestWorkflow_Next_AfterLiveSchemaEditDropsInFlightNode_MustNotPanic` — the one F-01 test
name that is stable; at doc-creation time it lives in `internal/services/f01_3_verification_test.go`
(verified by grep), and the F-01 test set was still being re-shaped for fix-agnosticism, so re-locate it
by name rather than by path.

Both cited lines were re-confirmed by reading while writing this dossier: `workflow.go:312` is
`currentNode, _ := w.graph.FindNode(currentAuditEntry.FunctionNodeID)` and `:314` is
`switch len(currentNode.OutputEdges())`, whose receiver method is `node.go:67-68`
(`func (n *Node) OutputEdges() []*Edge { return n.outputEdges }`) — the deref site.

Open question the plan must answer, not the verifier: what *should* happen. "Fail the run" is the
finding's framing, but `Next` returns an `Action` with no error channel, so the fix has to choose
between a failure action, an error return that changes every caller, or a guard at the audit-entry
lookup one line above (`:311`, which also discards its error).

## Code area

| Path | Symbol / line | Why it matters |
| --- | --- | --- |
| `internal/workflow/workflow.go` | `:309` `Next(threadID)`, `:311` discarded `auditLog.Get` error, `:312` discarded `FindNode` error, `:314` deref | the defect and its immediate neighbour |
| `internal/workflow/node.go` | `:67-68` `OutputEdges()` | where the panic surfaces |
| `internal/repositories/graph_memory.go` | `FindByID` | one route to a missing node (A-06's subject) |
| `internal/repositories/postgres/workflow.go` | `loadGraph` | the other route — replay against a changed active schema (A-05's subject) |

## Route

Skip stage 1 — the premise is already executed with a named test. Go to
`fuse-implementation-planner` with the test name, the file:line facts above and the open question about
`Next`'s signature (invoke `fuse-engine-researcher` first only if the planner needs the full caller set
of `Next`). Then `fuse-go-implementer` — the existing test becomes the red commit, re-pointed at the new
expected behaviour — then `fuse-code-reviewer`, then the gates plus
`go test -race ./internal/... ./pkg/...`, since the panic kills an actor.

## Done when

*Derived — the finding carries no "Done when".*

- [ ] The maintainer has assigned this finding a tier and an id.
- [ ] A test drives `Next` with an audit entry whose node is absent from the graph and asserts **no
      panic**.
- [ ] The run reaches a defined terminal or failed state instead of taking down the workflow actor, and
      the chosen behaviour is asserted (not just "does not crash").
- [ ] The discarded error at `:311` (`auditLog.Get`) is handled too, or a written reason says why it
      cannot produce the same crash.
- [ ] The failure is observable — logged and/or journalled — so a missing node is diagnosable after the
      fact rather than silent.

## Out of scope

Do not pin runs to a schema version — **A-05** owns that, including the `schema_version` column and the
migration. Do not deep-copy the memory graph repository — **A-06** owns that. Either would *hide* one
route to this crash without fixing it; land this independently, as the finding says. Do not rework
`Next`'s threading or join semantics along the way (ADR-0011, ADR-0012 territory).

## Decisions

**Neither** an ADR nor a SpecKit spec if the fix is a guard plus a failure path on a `fix/` branch. It
becomes ADR-worthy only if the planner changes `Next`'s shape — that ripples through the actor callers
and touches **ADR-0011 · Thread model for fork/join and ForEach** and **ADR-0022 · Retry and
error-handling model** (which owns what "failing the run" means). `Workflow` is `internal/`, so no
changelog obligation — and there is no `CHANGELOG` file in this repo anyway. Next free ADR: **0035**.

## Log

- **2026-08-11** — Dossier created from BACKLOG_V2.md ("Found during verification — unfiled").
- **2026-08-11** — Raised by F-01.3 verification with executed evidence and deliberately **not fixed**,
  per "do not fix bugs found along the way". `workflow.go:312/314` and `node.go:68` re-confirmed by
  reading; the reproducing test currently sits in `internal/services/f01_3_verification_test.go`.
