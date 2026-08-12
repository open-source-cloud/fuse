# V-a · Agent tool calls are invisible to OTel *and* Prometheus

|  |  |
| --- | --- |
| **Tier** | Unfiled (found during verification) — **the maintainer must assign it a tier and an id** |
| **Size** | S (unfiled; sizing is the maintainer's call together with the id) |
| **Status** | Not started |
| **Premise** | READ — from control flow, not executed |
| **Depends on** | nothing to reproduce it. Any *fix* that reroutes agent tool calls is gated by the HARD GATE below |
| **Blocks** | nothing formally. It is standing evidence against **B-03**'s parity requirement, and relevant to **C-01** and **C-04** |
| **First agent** | `fuse-premise-verifier` |
| **Skill packs** | `capability-registry-and-agents`, `observability-tracing` |

## What the backlog says

> `internal/packages/agent_tools.go:86` (`InvokeTool`) calls `LoadedPackage.ExecuteFunctionSync`
> (`internal/packages/loaded_package.go:50`), which goes straight to the transport and never enters
> `internal/actors/workflow_func.go`. That file is the **only** creator of the `node.execute` span
> (`:90`) and the only caller of `recordNodeDuration` (`:181`). So every tool an agent calls produces no
> span and no `fuse_node_exec_duration_seconds` sample.
>
> This is direct evidence on B-03's hard design constraint — "an agent calling a capability and a node
> calling the same capability must produce the same journal entries, the same span shape and the same
> idempotency behaviour". Today they demonstrably do not. Distinct from the known gap that intercepted
> `system/*` functions have no node span.

## Premise status

**READ, not executed.** The control-flow claim was traced during F-01 verification and no test was run
against it. Two of the cited lines were re-confirmed by reading while this dossier was written:
`agent_tools.go:86` is `return pkg.ExecuteFunctionSync(functionID, execInfo)` inside `InvokeTool`, and
`workflow_func.go:90` is the `StartSpan(parentCtx, "node.execute", …)` call. Stage 1 still owes an
executed reproduction — a real OTel provider plus a real Prometheus registry, one agent tool call, and
an assertion that **zero** `node.execute` spans and **zero** `fuse_node_exec_duration_seconds` samples
were produced, next to a graph node call through the same function that produces both. That contrast is
what makes it a parity defect rather than a missing feature.

## Code area

| Path | Symbol / line | Why it matters |
| --- | --- | --- |
| `internal/packages/agent_tools.go` | `:79` `InvokeTool`, `:86` `ExecuteFunctionSync` | the agent path — bypasses the actor entirely |
| `internal/packages/loaded_package.go` | `:50` `ExecuteFunctionSync` | goes straight to the transport |
| `internal/actors/workflow_func.go` | `:90` `node.execute` span, `:181` `recordNodeDuration` | the node path — sole source of both signals |

## Route

1 · `fuse-premise-verifier` — hand it the two sub-claims (no span, no metric sample) plus the node-path
control. 2 · `fuse-engine-researcher` if the verdict is REPRODUCED: the fix spans the packages, actors,
tracing and metrics subsystems, so the planner would otherwise guess. 3 · `fuse-implementation-planner`.
4 · `fuse-go-implementer`. 5 · `fuse-code-reviewer`.

**Gate before step 3.** Rerouting agent tool calls through the actor path *is* an agent-execution-model
choice, and the backlog's HARD GATE forbids making one before F-02 is green. If the fix stays
instrumentation-only (emit the span and the metric at the agent call site), the gate does not bite; if it
proposes moving the call onto the node path, stop and say so.

## Done when

*Derived from the finding text — it carries no "Done when".*

- [ ] An executed test shows the current asymmetry: agent tool call → no `node.execute` span, no
      `fuse_node_exec_duration_seconds` sample; graph node call of the same function → both present.
- [ ] The maintainer has assigned this finding a tier and an id.
- [ ] After the fix, an agent tool call and a graph node call of the same function produce the same span
      shape and the same metric sample, asserted in one test that exercises both paths.
- [ ] The finding's status is reflected in B-03's parity constraint (it is the evidence that today's two
      paths diverge).

## Out of scope

Do not build the capability registry — **B-03** owns that. Do not add GenAI attributes (agent id, model,
tokens, cost) — **C-04** owns those, and it declares dependencies on A-08 and C-03. Do not fix the
separate, known gap that intercepted `system/*` functions have no node span. Do not touch the trace-context
breaks — **A-04** owns those, and its own scope correction (F-01.1) already makes it a public API change.

## Decisions

Possibly an ADR, and only if the fix changes *where* an agent tool executes rather than what it emits —
that is exactly the kind of costly-to-reverse choice ADRs exist for, and it is the choice the HARD GATE
defers. Existing decisions it touches: **ADR-0020 · Observability: metrics, tracing, execution traces**
(the span and metric conventions), **ADR-0007 · Agent reasoning loop & tools-from-functions** (why the
agent path exists at all), and **ADR-0005 · AI agents as workflow nodes**. Next free ADR number is **0035**.

## Log

- **2026-08-11** — Dossier created from BACKLOG_V2.md ("Found during verification — unfiled").
- **2026-08-11** — Raised by F-01 verification and deliberately **not fixed**, per the backlog rule "do
  not fix bugs found along the way". `agent_tools.go:86` and `workflow_func.go:90` re-confirmed by
  reading while writing this dossier; still no executed evidence.
