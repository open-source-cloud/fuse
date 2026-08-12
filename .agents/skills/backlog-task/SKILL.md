---
name: backlog-task
description: Run one task from the repo-root FUSE backlog end to end — locate and scope it, verify its premise, research, plan, implement test-first, review, gate and land — delegating each stage to the `fuse-*` agents. Use when the user types `/backlog-task`, names a task id (F-02, A-07, B-01, C-08 …), asks "what should I work on next", or starts any engine work the backlog owns.
---

# backlog-task

The master entry point for a working session on FUSE Core. It turns one backlog task into one merged PR through a fixed pipeline: **locate → verify → research → plan → implement → review → land**. Each stage delegates to a specialised agent; this skill decides *which* agent, *when*, and *what to hand it*.

Invoked with an id (`/backlog-task F-02`) it runs that task. Invoked bare (`/backlog-task`) it reads the dispatch order and current sprint and *proposes* the next task, then stops for confirmation — it never picks and starts in one move.

## When to use

- A backlog task id is named, or the user asks what to pick up next.
- A bug report arrives that the backlog already owns (check before opening an editor).
- Someone is about to "just fix" something the backlog cites — the premise has not been verified yet.

**Not** for: authoring an ADR alone (use `write-adr`), a pure research question (`fuse-engine-researcher` directly), or reviewing a change already built (`fuse-code-reviewer` directly).

## Ground truth

Read these; do not work from memory or from a summary of them.

| File | Authoritative for |
| --- | --- |
| `BACKLOG_V2.md` (repo root) | Task ids, tiers, sizes, dependencies, "Dispatch order" (§ near the end), sprints, the HARD GATE, and "Rules for the implementing agent" (final section) |
| `.agents/agents/*.md` | The pipeline agents' own contracts — input, output shape, hard boundaries |
| `.agents/rules/*.mdc` | 13 numbered coding rules (`01-`…`13-`) plus `README.mdc`; all 14 carry `alwaysApply: true` |
| `docs/adr/` | 33 ADRs (`0001`–`0033`), MADR 3.0, index in `docs/adr/README.md` |
| `.specify/memory/constitution.md` | Governing principles; I Test-First and II Quality Gates are NON-NEGOTIABLE |
| `AGENTS.md` | Learned workspace facts + the stage→agent map. Prose can drift from the tree; see Traps |

**Backlog naming.** The canonical queue is the repo-root file titled `# FUSE — product shape and backlog`, currently `BACKLOG_V2.md`. If the maintainer has renamed it to `BACKLOG.md`, that title identifies the canonical one. The older superseded backlog renumbered **every** id — never carry an id across files.

**Authoring location.** Guidance is authored under `.agents/` only. `.claude/{rules,skills,commands,agents}` and `.cursor/{rules,skills,commands}` are symlinks into it (ADR-0009) — writing through them corrupts the source of truth.

## The pipeline

### 0 · Locate and scope

1. Read the task's full entry from the repo-root backlog. Quote its body and "Done when" verbatim into your working notes — every later stage works from your restatement.
2. Restate **tier · size · dependencies** from the entry (sizes: S ≈ under a day, M ≈ a few days, L ≈ a week with design risk).
3. Check the "Dispatch order" graph and the sprint lists. Current sprints, verbatim from the backlog:
   - **Sprint 1 — prove the engine:** F-01, F-02, F-03, A-07, A-06.
   - **Sprint 2 — write the spec:** B-01, plus A-01, A-04, A-05 in parallel.
   - **Sprint 3 — the pillar:** B-02, B-05, then B-06.
4. **Refuse to start a task whose blockers are open**, and name them. Dependencies as declared in the backlog entries themselves (re-read the entry; do not trust this list alone): **F-02 blocks all of Tier A**; A-02→F-02; A-03→A-02, F-02, B-01; A-06→F-01.3; B-02→B-01, A-04, A-10; B-03→B-01; B-04→B-03; B-05→B-01; B-06/B-07→B-02, B-05; B-08→B-01; B-10→A-01; C-01→B-03; C-04→A-08, C-03; C-05→A-11; C-10→B-01; C-12→F-02; all of D-* → Tier A green. Non-dependency ordering notes the entries also carry: A-07 "do first"; A-11 "ADR before code"; C-02 "overlaps A-11(a) — do them together".
5. **HARD GATE:** *do not choose an agent execution model before F-02 is green.* If the task or the conversation is drifting toward that decision and F-02 has not landed, stop and say so.

Bare invocation stops here with a proposal: id, why it is next (dispatch order + sprint), its blockers' status, and its size.

### 1 · Verify the premise — the stage that can end the pipeline

Every task cites a *report about* the code, not the code. Delegate to **`fuse-premise-verifier`**. It writes and **executes** a failing test and returns `REPRODUCED` / `NOT-REPRODUCED` / `PARTIAL` / `BLOCKED` with verbatim output.

- `REPRODUCED` → continue to stage 2 or 3; that test is the first commit of the PR.
- `PARTIAL` → re-scope the task to the clause that reproduced and record which clause did not.
- **`NOT-REPRODUCED` → the pipeline ENDS here.** Close the task in the backlog with the verifier's evidence (file:line showing why the code does not do what the entry says). This is a legitimate, expected, valuable outcome — the backlog's own header says so.
- `BLOCKED` → supply what is missing (a DSN, Docker, a kill point) and re-run; do not proceed on an unverified premise.

Several premises were **read as diverging** from the code (see Traps) — read, not executed. Expect closures and re-scopes.

### 2 · Research

Only when the question spans subsystems, or when the plan would otherwise guess. Delegate to **`fuse-engine-researcher`** — read-only, returns hop-by-hop control flow with every claim marked `CONFIRMED-BY-READING` or `INFERRED`. Load the skill pack named in the routing table *before* reading code, so the patterns are in context.

Skip it for a single-file change whose premise the verifier already proved.

### 3 · Plan

Delegate to **`fuse-implementation-planner`**. It refuses to plan an unverified premise, so stage 1 must have run. Its plan is shown to the maintainer **before any production code is written**.

Decide the decision-record weight here:

| Situation | Instrument |
| --- | --- |
| Costly to reverse, shapes the architecture (A-03 delivery guarantee, A-11 output addressing, A-05 version pinning, B-03 registry) | **ADR** via the `write-adr` skill — MADR 3.0 from `docs/adr/template.md`, index row in `docs/adr/README.md` |
| New subsystem or public contract needing requirements + tasks (a new SDK surface, a whole capability registry) | **SpecKit** — `.agents/commands/speckit.*`, creates a `NNN-short-name` branch and `specs/NNN-*/` |
| An ordinary bug fix, a config guard, a plumbing task | **Neither.** A `feat/` or `fix/` branch and a PR body |

Full SpecKit ceremony is overkill for a backlog task; the two existing specs (`specs/001-ai-agent-node`, `specs/002-anthropic-provider`) are both feature-sized. Note SpecKit's `NNN-short-name` branch naming contradicts the `feat/`/`fix/` prefixes in `.agents/rules/11-development-workflow.mdc`: `.specify/scripts/bash/common.sh:check_feature_branch` errors out ("Not on a feature branch") on any branch that does not match `^[0-9]{3}-`.

**Protocol tasks invert the order.** If the task touches the remote-node protocol surface, the **spec comes first** — route to `fuse-protocol-spec-author` and land B-01 before any Go. B-01 explicitly "blocks every other B task · do this before writing any Go".

### 4 · Implement

Delegate to **`fuse-go-implementer`**, or **`fuse-e2e-harness-engineer`** for F-02-class work (real process, SIGKILL, restart, CI stage). The reproducing test lands **first, failing, in the first commit** — this is a constitution-level rule (Principle I), not a preference.

### 5 · Review

Delegate to **`fuse-code-reviewer`** (read-only; it reports, it never fixes). Loop back to stage 4 until every finding is resolved, or explicitly deferred with a written reason recorded in the PR body. A `BLOCKER` finding is never deferred.

### 6 · Gates and land

1. `make lint && make build && make test` — in that order, no exceptions (`.agents/rules/12-quality-gates.mdc`). `./scripts/pre-commit-gates.sh` runs the same three quietly.
2. Add whatever the gates do not cover: `go test -race ./internal/... ./pkg/...` for actor/concurrency work, `make test-functional` with `DB_POSTGRES_DSN` exported for anything persistence-shaped, `make e2e-local` for engine-boundary work.
3. Branch prefix `feat/ fix/ docs/ refactor/ test/ chore/`; Conventional Commits subjects (the release version is derived from them).
4. **One task, one PR.**
5. Public API change → changelog entry + migration path. **There is no `CHANGELOG` file in this repo today** — raise that with the maintainer rather than inventing one.
6. HTTP/API change → Swagger annotations + `make swagger`, then `docs/API.md`, then the README route summary only if the public surface changed (`docs/CONTRIBUTE.md`).
7. **Update the backlog**: mark the task done (or closed with evidence), and **append** every drive-by bug found — with file:line evidence — instead of fixing it.

## Agent invocation

| Agent | Invoke at | Pass it | Returns | Writes files? |
| --- | --- | --- | --- | --- |
| `fuse-premise-verifier` | Stage 1, always | The task id + the premise quoted verbatim, split into numbered sub-claims | `VERDICT`, `EVIDENCE` (test path, exact command, verbatim output, driver), `CODE FACTS`, `DIVERGENCE FROM BACKLOG`, `SUGGESTED NEXT ACTION`, `NEW BUGS FOUND` | Yes — test files only |
| `fuse-engine-researcher` | Stage 2, when the question spans subsystems | One question, decomposed into sub-questions | `FLOW` hop by hop, `CLAIMS` each marked `CONFIRMED-BY-READING`/`INFERRED`, `NEGATIVE RESULTS`, `DRIVER PARITY`, `DOC / ADR DRIFT`, `OPEN QUESTIONS` | **No** |
| `fuse-implementation-planner` | Stage 3, after a `REPRODUCED`/`PARTIAL` verdict — **but never first for B-01 or any wire-surface change; those start at `fuse-protocol-spec-author` (stage 3, "Protocol tasks invert the order")** | The task id **and** the verifier's full verdict block | `FILES`, `TESTS` (failing-first named), `PARITY`, `HA`, `MIGRATION`/`PUBLIC API`/`CHANGELOG`, `DECISION RECORD`, `PROTOCOL IMPACT`, `OUT OF SCOPE`, `DONE WHEN`, `GATE COMMANDS` | **No** |
| `fuse-go-implementer` | Stage 4 | The plan, verbatim | `STATUS`, `RED TEST` (test-only commit sha + verbatim failure), `CHANGES`, `GATE` (lint/build/test/test-functional/race, each PASS·FAIL·SKIPPED), `GREEN TEST`, `PARITY`, `PROTOCOL`, `OUT OF SCOPE HELD`, `BACKLOG ADDITIONS` | Yes |
| `fuse-e2e-harness-engineer` | Stage 4 for F-02-class work | The plan plus the scenario list from F-02 | `STATUS`, `RIG` (binary, DSN, object store, port), `FILES`, `SCENARIOS` (per scenario GREEN·RED·BLOCKED + owning task id + probe + assertion), `CI`, `ENGINE BUGS FOUND`, `NOT COVERED` | Yes |
| `fuse-code-reviewer` | Stage 5 | The diff scope (working tree / branch vs `main` / PR number) + the task id | `VERDICT`, `GATES`, `NOT COVERED BY GATES`, ranked `FINDINGS` each with a concrete scenario, `TEST-FIRST EVIDENCE`, `SCOPE`, `PROTOCOL CONTRACT`, `OBLIGATIONS`, `BACKLOG ADDITIONS` | **No** |
| `fuse-protocol-spec-author` | Before any Tier B Go | The protocol surface being touched | `STATUS`, `SPEC` (path + version), `DECISIONS` (all eight B-01 questions, none `TBD`), `ADR`, `ENGINE FACTS CITED`, `ENGINE GAPS THIS SPEC CREATES`, `UNBLOCKS`/`BLOCKS-UNTIL-PUBLISHED` | Yes — **Markdown only** (spec doc, ADR, ADR index row); never engine Go |

All seven were present under `.agents/agents/` when this file was written. Still run `ls .agents/agents/` before delegating — the roster is authored, not shipped, and a missing agent means doing that stage inline, not skipping it. The `Returns` column above is transcribed from each agent's own `## Output format` block; if an agent file is revised, re-sync this table from it rather than trusting this copy.

## Routing table — every task in the backlog

`Premise` column: `VERIFY` = report-derived, run stage 1 (the default for every task). `VERIFY ⚠` = the entry's wording diverges from what the code was read to do — see Traps. `CONFIRMED` = the cited code fact was read directly in this repo, but the *fix* still needs the failing test first. `net-new` = nothing to reproduce; it is new construction.

`Skill pack` column: prefer the **FUSE-specific** packs — `durable-execution-internals`, `crash-resume-testing`, `persistence-and-migrations`, `observability-tracing`, `remote-node-protocol`, `capability-registry-and-agents`, `function-package-authoring`. Each carries `name`/`description` frontmatter and names the backlog ids it owns; they describe *this* engine. The rest (`actor-model-patterns`, `ddd-patterns`, `clean-hexagonal-architecture`, `microservices-architecture`, `cqrs-event-driven`, `go-concurrency`, `go-senior-developer`) are generic, frontmatter-less reference material — reach for them only as a supplement.

### Tier F — foundation

| id | intent | primary code area | skill pack | premise |
| --- | --- | --- | --- | --- |
| F-01 | Verify three findings: discarded span ctx; duplicate subworkflow child on restart; shared graph pointer | `internal/actors/workflow_func.go:96`; `internal/workflow/workflow.go:findPendingThreads/replayPendingThread`; `internal/repositories/graph_memory.go:FindByID` | observability-tracing (.1), durable-execution-internals (.2), persistence-and-migrations (.3) | this task *is* stage 1 |
| F-02 | Crash-resume e2e harness: real process, real Postgres, fs store, SIGKILL, restart, assert | `tests/e2e/`, `docker-compose.yml` (`e2e` profile), `.github/workflows/e2e.yml` (jobs `e2e-fast`, `e2e-slow`); assertion target `internal/actors/workflow_handler.go:156` | crash-resume-testing + durable-execution-internals + `.agents/rules/07-testing.mdc` | CONFIRMED — `Resume()` has exactly one call site (`workflow_handler.go:156`) and no test caller |
| F-03 | Reject persistent DB + ephemeral object store in config validation | `internal/app/config/config.go:Validate`; `internal/app/di/repos.go`; `internal/app/di/objectstore.go` | persistence-and-migrations | CONFIRMED — `Validate` checks only the `CLUSTER_ETCD_ENDPOINTS` pairing and returns `nil` otherwise |

### Tier A — core correctness

| id | intent | primary code area | skill pack | premise |
| --- | --- | --- | --- | --- |
| A-01 | Input payload on trigger, exposed to `SourceFlow` under `trigger.*` | `internal/dtos/workflow.go`; `internal/messaging/trigger_workflow.go`; `internal/actors/workflow_sup.go:spawnWorkflowActor`; `internal/workflow/workflow.go:Trigger` | function-package-authoring + ddd-patterns | VERIFY |
| A-02 | Replay the intercepted system functions (subworkflow, awakeable, sleep) | `internal/workflow/workflow.go:replayJournalEntries`; `internal/actors/workflow_handler.go:handleSystemSleep`/`handleSystemWait`/`handleSystemSubWorkflow` | durable-execution-internals + actor-model-patterns | VERIFY ⚠ awakeable clause |
| A-03 | Decide + enforce replay semantics for pending remote steps (ADR) | `internal/workflow/workflow.go:findPendingThreads/replayPendingThread`; `internal/handlers/async_function_result.go` | durable-execution-internals + remote-node-protocol + write-adr | VERIFY ⚠ execID clause |
| A-04 | Trace context end to end; add `context.Context` to `ExecutionInfo` | `internal/actors/workflow_func.go:96`; `internal/tracing/provider.go:ExtractCarrier`; `pkg/workflow/execution_info.go`; `internal/handlers/trigger_workflow.go` | observability-tracing | CONFIRMED — `_ = nodeCtx` at `workflow_func.go:96` |
| A-05 | Pin runs to a schema version; leave in-flight runs alone on rollback | `internal/repositories/postgres/workflow.go:loadGraph`; `internal/repositories/postgres/migrations/`; `internal/workflow/versioned_schema.go`; `internal/services/graph_service.go` | persistence-and-migrations + write-adr | VERIFY |
| A-06 | Deep-copy from the memory graph repo; assert driver parity | `internal/repositories/graph_memory.go:FindByID`; `tests/functional/graph_repository_test.go:contractTestGraphRepository` | persistence-and-migrations | VERIFY (= F-01.3) |
| A-07 | Suppress journal append during replay via a replay flag | `internal/workflow/workflow.go:SetResultFor`; `internal/workflow/journal.go:LoadFrom`/`NewEntries` | durable-execution-internals | VERIFY ⚠ trace clause |
| A-08 | Attempt counter on the journal entry; rebuild `RetryTracker` on replay | `internal/workflow/retry_tracker.go`; `internal/workflow/workflow.go:HandleNodeFailure`; `internal/workflow/trace_builder.go` | durable-execution-internals | VERIFY |
| A-09 | Inline small payloads; content-address large ones | `internal/repositories/postgres/journal.go:Append/LoadAll`; `internal/repositories/postgres/workflow.go:Save`; `internal/repositories/postgres/trace.go`; `pkg/objectstore/` | durable-execution-internals + persistence-and-migrations | VERIFY |
| A-10 | Configurable pool size; remote calls must not hold a worker | `internal/actors/workflow_func_pool.go:40`; `internal/actors/workflow_instance_sup.go` | go-concurrency + actor-model-patterns (+ remote-node-protocol for the non-blocking half) | CONFIRMED — `PoolSize: 3` hardcoded in `WorkflowFuncPool.Init` |
| A-11 | Address node output by execution — **ADR before code** | `internal/workflow/workflow.go:428` (`w.aggregatedOutput.Set(entry.FunctionNodeID, …)`); `pkg/store/kv.go` | capability-registry-and-agents + durable-execution-internals + write-adr | CONFIRMED — keyed by node id, not exec id |

### Tier B — extension protocol and SDKs

**Nothing in Tier B is shipped behaviour.** The backlog states it plainly: "everything here is net-new". The engine today declares `HTTP` and `gRPC` in `pkg/transport/type.go` and implements neither; `PackagedFunction.Function` is `json:"-"`, so an API-registered function is metadata-only and `LoadedPackage.ExecuteFunction` returns `function %s has no transport`. The `primary code area` column below names *where the proposed work would land, or the existing raw material it builds on* — never a capability you can use now. Do not describe any of it to a user as something FUSE does.

| id | intent (**all proposed**) | primary code area / raw material | skill pack | premise |
| --- | --- | --- | --- | --- |
| B-01 | **Spec document** for the remote node protocol — blocks every other B task | proposed new doc under `docs/`; raw material in `pkg/transport/type.go`, `internal/packages/transport/function.go`, `pkg/workflow/execution_info.go`, `internal/handlers/async_function_result.go`, `pkg/workflow/fn_result.go:NewFunctionResultAsync` | remote-node-protocol + write-adr | net-new (spec, not a defect) |
| B-02 | Implement the HTTP transport; validate the callback's exec is pending | `internal/packages/loaded_package.go:41` and `:MapToRegistryPackage`; `internal/packages/transport/`; `internal/packages/loaded_function.go` | remote-node-protocol + function-package-authoring | CONFIRMED (of the *gap*, not the feature) — `loaded_package.go:41` is the "has no transport" dead end; `async_function_result.go:HandlePost` does no pending-state or duplicate validation |
| B-03 | Capability registry: one registration → node **and** tool projection | `internal/packages/registry.go`; `internal/packages/agent_tools.go` (today an adapter over the same `Registry`, not a second projection); `internal/services/package_service.go` | capability-registry-and-agents + remote-node-protocol | net-new construction; VERIFY the projection-parity premise against `agent_tools.go` first |
| B-04 | MCP ingestion — import an MCP server's tools as capabilities | proposed new package under `internal/packages/`; `internal/dtos/package.go` | capability-registry-and-agents | net-new |
| B-05 | SDK conformance suite, built before the first SDK | proposed new test tier; model it on `tests/e2e/` + `.github/workflows/e2e.yml` | remote-node-protocol + crash-resume-testing | net-new |
| B-06 | SDK: TypeScript / Node (plain + NestJS) | **outside this repo** — built against the B-01 spec and the B-05 suite | remote-node-protocol (spec side only) | net-new |
| B-07 | SDK: PHP / Laravel | **outside this repo** — built against the B-01 spec and the B-05 suite | remote-node-protocol (spec side only) | net-new |
| B-08 | Worker registration lifecycle: register, health, deregister, drain | `internal/services/package_service.go`; `internal/actors/mux_worker.go` (the HTTP route table — where lifecycle routes would be declared); `internal/repositories/postgres/package.go` | remote-node-protocol | net-new; VERIFY what registration exists today before designing around it |
| B-09 | Guard sub-workflow recursion: depth counter, max, cycle detection | `internal/actors/workflow_handler.go:handleSubWorkflowAction`; `internal/workflow/subworkflow.go` | durable-execution-internals (the `subworkflow:*` entries and the spawn path) + actor-model-patterns | VERIFY (the absence of a guard) |
| B-10 | Pass input into sub-workflows; add an output selector | `internal/actors/workflow_handler.go:handleSubWorkflowAction`; `internal/messaging/trigger_workflow.go`; `internal/workflow/workflow.go:AggregatedOutputSnapshot` | actor-model-patterns + durable-execution-internals | VERIFY |

### Tier C — agents in Core

**Tier C is proposed work too.** `ai/agent` and `ai/chat` exist as async node functions, but every capability named below — registry binding, first-class memory, a model gateway, GenAI cost on the trace, a typed human-gate payload, durable timers, an awakeable-id route, `GET /v1/workflows/{id}/state` — is a task, not a feature. Where the premise column says `CONFIRMED`, what is confirmed is the **gap** (a symbol that exists with no caller, a route that is absent), never the proposed replacement.

| id | intent (**all proposed**) | primary code area / raw material | skill pack | premise |
| --- | --- | --- | --- | --- |
| C-01 | Bind the agent node to registry capabilities; identical journal + trace | `internal/packages/functions/ai/agent.go`, `tools.go`; `internal/packages/agent_tools.go` | capability-registry-and-agents | VERIFY |
| C-02 | Memory as a first-class concept (working / episodic / semantic) | `internal/packages/functions/ai/context.go`; proposed new repository | capability-registry-and-agents + write-adr | net-new; backlog says "overlaps A-11(a) — do them together" |
| C-03 | Model gateway: provider routing, budgets, redaction | `internal/llm/providers/` (`anthropic`, `openaicompat`) | capability-registry-and-agents + microservices-architecture | net-new |
| C-04 | GenAI telemetry and cost on the trace | `internal/packages/functions/ai/usage.go`; `internal/tracing/provider.go`; `internal/metrics/` | observability-tracing | VERIFY · declared deps A-08 + C-03 |
| C-05 | Human gate payload schema + correction semantics | `internal/workflow/awakeable.go`; `internal/packages/functions/system/wait.go`; `internal/handlers/resolve_awakeable.go` | capability-registry-and-agents + durable-execution-internals | net-new; the entry declares it depends on A-11 being resolved |
| C-06 | Durable timers and an expiry sweeper; re-arm on boot, claim-aware | `internal/actors/execution_timer.go`; `internal/actors/workflow_handler.go`; `internal/actors/workflow_claim_actor.go`; `Awakeable.DeadlineAt` (`internal/workflow/awakeable.go:31`) | durable-execution-internals + actor-model-patterns + go-concurrency | VERIFY |
| C-07 | Expose awakeable IDs over the API | `internal/repositories/awakeable.go:16` (`FindPending`); `internal/actors/mux_worker.go`; `internal/handlers/get_workflow.go` | durable-execution-internals + clean-hexagonal-architecture | CONFIRMED — `FindPending` is declared on the interface and implemented by both drivers; no route in `mux_worker.go` exposes it |
| C-08 | ForEach state survives restart; stop the silent interception bypass | `internal/workflow/foreach.go`, `foreach_state.go`; `internal/actors/workflow_handler.go:spawnForEachBatch` (line 1013) | durable-execution-internals + persistence-and-migrations + actor-model-patterns | CONFIRMED — plus a blocking enum gap, see Traps |
| C-09 | Mid-flight run read model (proposed `GET /v1/workflows/{id}/state` — no such route today) | `internal/handlers/get_workflow.go`, `get_workflow_snapshot.go`; `internal/workflow/execution_snapshot_builder.go`; `internal/events/memory_bus.go` | durable-execution-internals + cqrs-event-driven | VERIFY |
| C-10 | Per-node idempotency key from `(runID, execID, attempt)` | `internal/idempotency/store.go`; `pkg/workflow/execution_info.go` | remote-node-protocol | VERIFY · declared dep is B-01; that the `attempt` component must come from A-08 is inference, not a backlog-declared dependency |
| C-11 | Claim failure must fail closed | `internal/actors/workflow_handler.go:369` (`claimForThisNode`); `internal/repositories/postgres/claim.go`; `internal/actors/workflow_claim_actor.go` | actor-model-patterns + persistence-and-migrations | CONFIRMED — on a claim-store error it logs "running anyway" and returns `true` |
| C-12 | Multi-thread resume correctness | `internal/workflow/workflow.go:226` (`buildResumeAction`) | durable-execution-internals + crash-resume-testing | VERIFY ⚠ backlog itself flags it "premise is inference" |

### Tier D — Enterprise layer

`D-01` tenancy/RBAC/SSO · `D-02` console · `D-03` collaboration · `D-04` cost dashboards · `D-05` audit export · `D-06` packaging.

**All D-* are Enterprise repo — not implemented in `core/`.** They start once Tier A is green and consume Core's public API only. If a task in this repo only makes sense with the commercial layer present, it belongs in the commercial layer, not here.

## Non-negotiable pipeline invariants

Restated from the backlog's "Rules for the implementing agent". These are not guidance; a stage that violates one is wrong.

1. **Verify before fixing.** Every task cites a report, not the code. If the premise does not hold, close the task and report it — that is a success.
2. **One task, one PR.** They are deliberately separable.
3. **A test reproducing the bug lands before the fix**, in the same PR, failing in the first commit.
4. **Do not fix bugs found along the way.** Append them to the repo-root backlog with evidence and keep going.
5. **The protocol spec is a contract.** Once B-01 is published and an SDK exists, changing it breaks strangers. Treat it with more care than engine internals — spec first, code second, versioned.
6. **Core stays usable without Enterprise.**
7. **Public API changes need a changelog entry and a migration path.** Apache-2.0, with users who are not you.

Plus, from this repo: gates run **lint → build → test** in that order; author under `.agents/` only; never widen the quarantined `"untriggered"` retry in `tests/e2e/workflow_fixture_e2e_test.go` to make a suite green.

## Traps — what looks right and is wrong

Each **code fact** below was read in this repo at the cited symbol. The **runtime consequences** drawn from those facts ("each restart adds a row", "two callbacks race") are reasoning over the control flow, not observations — nothing here has been executed under a real SIGKILL, because F-02 is what would make that possible. Treat the code facts as load-bearing and the consequences as the hypothesis stage 1 must confirm. They are the reason stage 1 exists.

- **A-03's execID clause is contradicted by the code.** The backlog says the worker is re-invoked "with a new execID, and the original callback is orphaned". Read at `internal/workflow/workflow.go:295`: `replayPendingThread` sets `FunctionExecID: workflow.ExecID(pt.execID)` — the **original** execID carried out of the journal by `findPendingThreads`, not a new one — so the pre-crash callback URL still routes. *Inferred:* the actual exposure is a duplicate dispatch with two callbacks able to arrive for one execID, since `internal/handlers/async_function_result.go:HandlePost` (read in full) does no pending-state, duplicate or staleness validation — it binds the path params and sends the message. Re-scope before implementing.
- **A-02's awakeable clause looks wrong as written.** The backlog says the token you handed out "is orphaned pending forever". Read: `internal/handlers/resolve_awakeable.go:66` resolves via `awakeableRepo.FindByID(awakeableID)` and then sends the awakeable's *own stored* `ExecID`/`ThreadID`, so the pre-crash token still routes after a restart. What the code does show is `handleSystemWait` (`workflow_handler.go:727`) minting `awakeableID := uuid.New().String()` on **every invocation**, with no read of `JournalAwakeableCreated` anywhere. *Inferred from that:* since resume re-issues pending steps and `system/wait` is intercepted, each restart should add an extra awakeable row plus an extra `awakeable:created` entry and re-arm the full original timeout. Verify both halves — the failing clause and the replacement — before re-scoping.
- **A-07's trace clause is contradicted by the code.** The backlog says `BuildTrace` "renders each step N+1 times after N restarts". Read at `internal/workflow/trace_builder.go:10`: a step row is appended **only** on `JournalStepStarted`, and `replayJournalEntries` (`workflow.go:199`) does not append a journal entry for that type — it only calls `auditLog.NewEntry`. It is `JournalStepCompleted` that routes into `SetResultFor`, which *does* append. So a duplicated `step:completed` finds the existing `stepIdx[execID]` and mutates that row in place: `CompletedAt`/`Duration` recomputed against the **replay** timestamp and `Status` forced back to `completed`. Corruption in place, not duplication. The journal-growth half of the premise stands. Critical constraint on the fix: line 428's `w.aggregatedOutput.Set(...)` inside `SetResultFor` is the only thing rebuilding aggregated output during replay — gate the append, never skip the call.
- **C-12 is narrower than written.** Read at `internal/workflow/workflow.go:226`: `buildResumeAction` returns the first non-noop action **only on the zero-pending branch**; with one pending thread it replays that thread, and with several it fans out correctly into a `RunParallelFunctionsAction`. So the reproducing scenario is specifically a crash at a fan-out/join boundary with nothing still in flight — not "any multi-thread crash". `Next()` also mutates state and appends `thread:finished`, so actions discarded by that loop have already had side effects; *inferred from that*, the bug may present intermittently. Do not encode that guess in the test — let the F-02 multi-thread scenario settle it.
- **C-08 has a blocking prerequisite the entry does not mention.** Confirmed by reading both sides: `internal/workflow/journal.go` declares 18 `JournalEntryType` constants including the four `foreach:*` values (lines 43–49), while `migrations/000001_create_tables.up.sql:12` creates `journal_entry_type` with **13** values and no `foreach:*` among them; the only later widening is `000004_add_manual_retry_journal_type.up.sql`, which adds `step:manual-retry` alone. *Inferred (not executed):* with `DB_DRIVER=postgres`, inserting a `foreach:*` entry must therefore be rejected by the enum, so a ForEach run's journal flush fails. Ship the `ALTER TYPE … ADD VALUE` migration first — verify this under a real Postgres as the very first step of C-08, since nothing about ForEach replay is observable until it holds.
- **A green `make test` proves less than it looks.** No `-race` in the Makefile or in `.github/workflows/ci.yml`. `make test-functional` **skips silently** without `DB_POSTGRES_DSN` (`tests/functional/postgres_test.go:29` calls `t.Skip`). E2E is a separate `workflow_run` workflow gated on CI success, and its `e2e-slow` job additionally requires `head_branch == 'main'` — so the slow tier only ever runs after merge. `scripts/check-coverage.sh` is referenced by neither the Makefile nor any workflow. `.git/hooks/` holds no installed hook in this working copy.
- **A fresh clone cannot build.** `docs/docs.go` and `docs/swagger.{json,yaml}` are gitignored but blank-imported by `internal/actors/mux_server.go`. Run `make swagger` first, or `make build` *and* `make test` both fail.
- **Never validate a resume fix against the memory driver.** `MemoryWorkflowRepository.Get` returns the stored `*Workflow` pointer as-is, with whatever threads/audit log/aggregated output the live object accumulated; `postgres/workflow.go:Get` calls `workflow.New(...)` and restores only `state`, so it hands back a fresh object. A resume test that passes under memory proves nothing about Postgres. Postgres also carries `CREATE UNIQUE INDEX idx_journal_wf_seq ON journal_entries (workflow_id, sequence)` (migration `000001`); memory has no equivalent.
- **Trust `mux_worker.go` and the `Makefile` over prose.** Two facts that older docs got wrong, both re-checked here: example seeding is `./bin/fuse seed examples --ci` (flag declared at `internal/app/cli/seed.go:56`) — there is no `examples-ci` make target and no `scripts/run-example-workflows.sh`. And `GET /v1/workflows/{workflowID}/status` **is not a route**: `internal/actors/mux_worker.go:220` registers `/v1/workflows/{workflowID}`, whose handler returns `dtos.GetWorkflowResponse` — `{workflowId, status}` and nothing else. Richer mid-flight state is C-09.
- **`.golangci.yml` declares `version: 2` but carries v1 schema keys.** Confirmed by reading: the file opens with `version: 2` and still uses `linters-settings:` (with `gocyclo.min-complexity: 15` and `govet.check-shadowing`), `run.skip-dirs`, `issues.exclude-rules` and `output.format` — all of which v2 renamed or moved. **Inferred, not observed** (golangci-lint was not executed to check): v2 therefore does not apply them, so the documented complexity ceiling and the "no errcheck/gosec on tests" exclusion are probably not in force. Write tests defensively (`defer func() { _ = f.Close() }()`), and do not casually "fix" the config mid-task — repairing it likely surfaces a wave of findings across the tree, which is a separate task.

## Exact commands

```bash
make swagger                                   # FIRST on a fresh clone; generates the gitignored docs package
make lint && make build && make test           # the mandated gate, in this order
./scripts/pre-commit-gates.sh                  # same three, quiet
go test -v -run TestName ./internal/workflow/  # single test
go test -race ./internal/... ./pkg/...         # NOT in any gate — run it for actor/concurrency changes
go clean -testcache                            # replay tests are order/timing sensitive

make infra-up                                  # docker compose --profile infra (PG + S3 + etcd)
DB_POSTGRES_DSN='postgres://fuse:fuse@localhost:5432/fuse?sslmode=disable' make test-functional
make migrate                                   # build, then ./bin/fuse migrate
make e2e-local                                 # build image, compose --profile e2e, tags=e2e against :9091, tear down
make ha-up / make ha-down                      # 3-node HA cluster from source
make test-benchmark

ls docs/adr/[0-9][0-9][0-9][0-9]-*.md | sed -E 's#.*/([0-9]{4})-.*#\1#' | sort -n | tail -1   # next ADR number
ls .agents/agents/                             # which pipeline agents are installed
```

Docker profiles are `infra`, `ha`, `e2e` in the single `docker-compose.yml`.

$ARGUMENTS
