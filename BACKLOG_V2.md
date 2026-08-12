# FUSE — product shape and backlog

Replaces the previous backlog, which was written for a separate TypeScript platform sitting on top of the engine. That framing is gone. Everything executes inside FUSE.

Derived from the Phase 0 capability report (`core/ @ 31c43e1`). Tasks cite the evidence they came from. **The report describes the code; it is not the code.** Task F-01 exists to verify before anything is built. If a premise fails verification, close the task and say so.

---

## 0. Product shape

**FUSE Core** — Apache-2.0, Go. The engine: graph execution, durable journal, agents, the extension protocol, and the SDKs that speak it. This is the thing people adopt, extend, and talk about.

**FUSE Enterprise** — commercial layer on top of Core. Console, tenancy, RBAC/SSO, audit, cost accounting, collaboration, packaging. No fork, no divergent engine: Enterprise is a consumer of Core's API like any other.

Two topologies, and the second is what makes the first credible:

```
Most customers:        Enterprise ──▶ Core

Customers who extend:  Enterprise ──▶ Core ──▶ their API
                                               (a node, via SDK)
```

**The extension protocol is the product pillar, not a feature.** An engine that only runs Go functions is a Go tool. An engine where a Laravel shop registers an existing endpoint as a node in twenty minutes is infrastructure. The async primitives already exist in Core (`NewFunctionResultAsync`, `POST /v1/workflows/{workflowID}/execs/{execID}`) and no node uses them; the transport layer declares HTTP and gRPC and implements neither. The protocol is half-built and undocumented. Finishing and specifying it is Tier B and it is the highest-leverage work in this document.

**One capability, two projections.** A registered external API must produce *both* a node (callable from a workflow graph) and a tool (callable by an agent) from a single definition. Registering the same endpoint twice, once per consumer, is the obvious mistake and it is very hard to undo later. Design the registry first, project from it second.

### What this decision costs

Running agents inside the engine means the conversation lives in the journal, so three things that were previously avoidable are now unavoidable core work: payload write amplification (A-09), pool blocking under long LLM calls (A-10), and per-execution addressing of node output (A-11). LLM latency also moves onto the engine's critical path. That is the price of one system of record, one journal, one trace, one product — the right trade for the positioning, but a real price, and it lands in Tier A rather than being deferred.

---

## Tiers

- **F — Foundation.** Verification and the safety net. Nothing else starts, and no execution-model decision gets made, until these pass.
- **A — Core correctness.** The engine's durability claims have to be true before anything is sold on them.
- **B — Extension protocol and SDKs.** The product pillar.
- **C — Agents in Core.** What makes it an agent engine rather than a workflow engine.
- **D — Enterprise layer.** Separate repo, separate lifecycle, starts once A is green.

Sizes: **S** ≈ under a day, **M** ≈ a few days, **L** ≈ a week or more with design risk.

---

## Tier F — Foundation

### F-01 · Verify the three high-leverage findings
**S · no dependencies**

1. `internal/actors/workflow_func.go` — node span context discarded (`_ = nodeCtx`), so function code cannot create child spans or inject `traceparent` outbound. **Now critical**, because it is the seam every SDK crosses.
2. Restart with a pending `system/subworkflow` spawns a second child — `findPendingThreads` matches `step:started` without completion, `replayPendingThread` re-issues, interception fires again.
3. `graph_memory.go:FindByID` returns the shared `m.graphs[id]` pointer live `Workflow`s hold, so `updateVersioned → graph.UpdateSchema` rewrites a running run's topology in place.

**Done when:** each is a checked-in failing test on `main`, or a written note that it did not reproduce and why.

### F-02 · Crash-resume e2e harness
**M · blocks all of A**

No test kills the process mid-run and asserts resume. `Workflow.Resume()` is called from one place (`workflow_handler.go:156`) with no unit test. Everything FUSE claims rests on this path.

Real process, real Postgres, fs object store, SIGKILL at a controllable point, restart, assert. Not in-process — `workflow_resilience_suite_e2e_test.go` covers in-process failure, which is a different thing.

Scenarios, each its own test: kill after a completed step; kill with a pending async node; kill with a pending `system/subworkflow` (**exactly one child after restart**); kill with a pending `system/sleep`; kill with a pending awakeable (pre-crash token still resolves); kill a multi-thread run between thread completions; three restarts on one run with stable journal length and trace.

**Done when:** it runs in CI and each scenario is green or has a linked task. Several will be red. That is the point — do not fix them here.

### F-03 · Config validation for driver combinations
**S**

`DB_DRIVER=postgres` with default `OBJECT_STORE_DRIVER=memory` yields durable journal rows whose `input_ref`/`result_ref`/`data_ref` point into a store that dies with the process. `LoadAll` then fails on every payload fetch and nothing is recoverable. Silent, and it looks correct until the first restart.

Extend `Config.Validate` to reject persistent-DB + ephemeral-object-store and any other mismatched pair, naming both variables in the error.

---

## Tier A — Core correctness

### A-01 · Input payload on workflow trigger
**S**

`dtos.TriggerWorkflowRequest` is `{schemaID, idempotencyKey, environment}` — no payload. Plumbing is built and dropped: `TriggerWorkflowMessage.Input` exists, `NewTriggerWorkflowWithInputMessage` exists, webhook/cron/event populate it — but `spawnWorkflowActor` takes three args, `WorkflowHandlerInitArgs` has three fields, and `Trigger` hardcodes `Args: map[string]any{}`.

Thread it end to end and expose it to `SourceFlow` under a reserved key (`trigger.*`).

**Elevated in this design:** when an agent composes a workflow run, parameterising the trigger *is* the composition mechanism. Without it the agent has no way to pass anything into what it starts.

### A-02 · Replay the intercepted system functions
**M · depends on F-02**

`replayJournalEntries` handles five of eighteen journal types. `JournalAwakeableCreated`, `JournalSleepStarted`, `JournalSubWorkflowStarted` each have one write site and **zero read sites**.

- **subworkflow** — restart spawns a duplicate child. Real duplicate side effect.
- **awakeable** — restart re-mints via `uuid.New()`; the token you handed out is orphaned pending forever.
- **sleep** — restart restarts the full duration.

All three are blocking now: agents inside the engine will use sub-workflows for composition and awakeables for gates.

### A-03 · Replay semantics for pending remote steps
**M · depends on A-02, F-02, B-01**

`findPendingThreads` matches `step:started` without completion — the shape of an in-flight remote call — and re-issues it. After a restart the SDK worker is invoked a second time, with a new execID, and the original callback is orphaned.

Decide and enforce, in an ADR: **(a)** at-least-once with a stable idempotency key the SDK dedupes on, or **(b)** replay-aware re-arming of the original execID.

The SDK contract makes (a) far more palatable than it was when the caller was hand-written — dedup can live in the SDK, not in every customer's controller. Whichever you pick, **write it in the protocol spec**, because every SDK author depends on it.

### A-04 · Trace context end to end
**M**

Three breaks. Inbound: `trigger_workflow.go:HandlePost` never reads headers, `startRootSpan` uses `context.Background()`; `ExtractCarrier` exists and is never called at the HTTP boundary. Handler→worker: works, add a regression test. Outbound: `_ = nodeCtx` discards the span context and `ExecutionInfo` has no `context.Context`, so function code cannot create children or inject `traceparent`.

**This is the seam the SDKs cross.** A distributed engine whose selling point is traceability, that breaks the trace at exactly the boundary it invites you to extend across, is unsellable. Add the context to `ExecutionInfo`; B-02 depends on it.

### A-05 · Pin runs to a schema version
**M**

`workflows` has `schema_id` and no version column; `postgres/workflow.go:Get → loadGraph(ctx, schemaID)` reads the *active* definition, so a run recovered after an edit replays against current topology. Rollback writes a new active version, silently swapping in-flight runs, unrecorded in their journals.

`FindByIDAndVersion` already exists on the service and both repositories — the run just never records where it started. Add `schema_version` to `workflows`, record and journal it at trigger, resolve `loadGraph` by `(schemaID, version)`, and leave in-flight runs alone on activate and rollback.

### A-06 · Stop the shared-pointer live edit
**S · depends on F-01.3**

Memory driver returns the live pointer; postgres rebuilds per call. Return a deep copy from memory and assert both drivers behave identically. A-05 fixes the durable half of this; A-06 fixes the live half. Both are needed.

### A-07 · Fix duplicate journal append on replay
**S · do first**

`replayJournalEntries` calls `SetResultFor`, which appends. `LoadFrom` sets `lastPersisted = seq`, so replay duplicates get fresh sequences and are persisted at the end of `Init`. The journal grows by a full copy of completed steps every restart and `trace_builder.go:BuildTrace` renders each step N+1 times after N restarts.

Suppress journaling during replay via a replay flag. It compounds, and it corrupts the trace — the thing you sell.

### A-08 · Distinguish retry attempts
**S**

Automatic retries reuse the same `ExecID` (`HandleNodeFailure → RetryFunctionAction{RunFunctionAction{FunctionExecID: execID}}`); attempts are separable only via `step:retrying` and the in-memory `RetryTracker`, not rebuilt on replay. Manual retry does it right (`RetryNode` mints a fresh ExecID, records `previousExecId`).

Add an attempt counter to the journal entry and rebuild `RetryTracker` on replay. Cost attribution per attempt (C-04) and the trace view both need it, and remote nodes make retries far more common.

### A-09 · Payload write amplification
**M · promoted — agents now put conversations in the journal**

Every payload-carrying journal entry is one object-store PUT. No threshold, no inlining, no compression, no dedup. A 200 KB transcript is re-PUT on every `step:started` whose input includes it and every `step:completed` whose result includes it, plus into `output.json` on every `Save`. At 50 concurrent runs × ~12 steps: ~600 PUTs, ~120 MB per round. `LoadAll` re-fetches the whole payload graph on every resume, and A-07's duplicates make each restart heavier than the last.

Two independent changes: **inline small payloads** in the PG row below a threshold; **content-address large payloads** by hash so an unchanged transcript is stored once and referenced N times. The second alone removes most of the amplification for conversational workloads.

### A-10 · Configurable pool size and non-blocking remote calls
**S–M · promoted**

`WorkflowFuncPool` is `PoolSize: 3` per instance. Any blocking node caps a run at three concurrent branches; one 90-second LLM or remote call holds a third of that run's capacity. Make it configurable globally and per-schema, and make sure the remote transport (B-02) never holds a worker while awaiting a callback.

### A-11 · Address node output by execution
**L · promoted from post-pilot — the biggest design risk in this document**

`SetResultFor` writes `w.aggregatedOutput.Set(entry.FunctionNodeID, ...)` — keyed by **node ID, not exec ID**. A node that runs twice overwrites its own output; `SourceFlow` only ever sees the latest. The journal keeps every execution; the data plane does not.

With agents inside the engine this stops being theoretical: an agent loop is a cycle, every turn re-executes the same node, and each turn destroys the previous one's output in the data plane. Conversation history, per-turn selections, and any "reject step 2, keep 1, 3 and 4" correction are all unaddressable from inside the graph.

**Do not start by re-keying the KV.** Two options, and the cheaper one is probably also the better one:

- **(a) Turn state as a first-class Core concept** — a dedicated store (repository + journal entry types) that agent nodes read and append to, addressed by `(runID, agentNodeID, turn)`, entirely separate from `aggregatedOutput`. Smaller blast radius, no breaking change to existing schemas, and it is a thing you want to exist anyway (C-02).
- **(b) Re-key `aggregatedOutput` by `(nodeID, execID)`** with a `latest` alias for compatibility and mapping syntax for a specific execution. Correct and general; breaking-ish; needs its own ADR and a migration story.

Decide in an ADR before either. This is the one item that is expensive to reverse.

---

## Tier B — Extension protocol and SDKs

The pillar. Everything here is net-new and it is what turns Core from a Go library into a platform.

### B-01 · Specify the remote node protocol
**M · blocks every other B task · do this before writing any Go**

A written spec, versioned, in the repo, that an SDK author who has never read the engine can implement. Existing raw material: `NewFunctionResultAsync`, `POST /v1/workflows/{workflowID}/execs/{execID}`, `pkg/transport/type.go` constants, `ExecutionInfo{WorkflowID, ExecID, Environment, Input, Finish}`. None of it is documented as a contract, and `PackagedFunction.Function` is `json:"-"`, so API-registered functions are metadata-only today (`MapToRegistryPackage` → "function %s has no transport").

The spec must settle:

1. **Push or pull.** FUSE calls the worker's HTTP endpoint, or the worker polls/streams from FUSE. Push fits NestJS and Laravel idiomatically — they are already HTTP servers — and fits "register your existing API as a node" exactly. Pull works behind firewalls with no inbound reachability, which is how Temporal survives enterprise networks. **Recommendation: push first**, with pull as a later transport for locked-down deployments. Decide explicitly; it is the hardest thing to change later.
2. **Invocation envelope** — `workflowId`, `execId`, `nodeId`, `threadId`, attempt, idempotency key, deadline, environment, input, callback URL, `traceparent`.
3. **Ack semantics** — sync result vs `202 + async callback`. The async path is the interesting one and the reason this exists.
4. **Completion** — success, business error (routes to the error edge), infrastructure error (retryable), and the distinction between them. Getting this taxonomy right is what makes retries safe.
5. **Delivery guarantee** — the A-03 decision, stated plainly, with the idempotency key SDKs dedupe on.
6. **Auth** — how a worker proves identity and how FUSE proves it is FUSE. Both directions, since customers will run this across trust boundaries.
7. **Capability declaration** — input/output schema, so a registered node is typed and `applyFlowMapping`'s coercion (`internal/typeschema/parse.go`) still works.
8. **Versioning** — of the protocol itself and of a registered capability.

### B-02 · Implement the remote transport in Core
**M · depends on B-01, A-04, A-10**

Make the HTTP transport real: `MapToRegistryPackage` stops registering non-internal functions metadata-only; a remote node dispatches per B-01, returns `NewFunctionResultAsync()`, releases its pool worker, and resumes on callback. Inject `traceparent`. Validate that the callback's exec is genuinely pending and reject duplicates and stale execIDs.

This is the generalised, specified version of what the previous backlog called "a generic async HTTP node". Same code, promoted from workaround to product surface.

### B-03 · Capability registry
**M · depends on B-01**

One registration produces two projections: a **node** usable in a graph, and a **tool** usable by an agent. Single definition, single schema, single auth config, single version. Registration via API and via schema-as-code.

Design constraint for the ADR: an agent calling a capability directly and a workflow node calling the same capability must produce the same journal entries, the same span shape, and the same idempotency behaviour. If the two paths diverge, traceability becomes conditional on which one was used — and that is the property Enterprise is sold on.

### B-04 · MCP ingestion
**M · depends on B-03**

Import an MCP server's tools as capabilities. Real customer flows already speak MCP — the reference broker assistant is entirely MCP tools returning `{items, systemMessage, userMessage}` — so this is how existing agent estates arrive on FUSE without a rewrite.

Consider the inverse (**expose** FUSE capabilities as an MCP server) but do not build both at once; ingestion is the one with a customer behind it.

### B-05 · SDK conformance suite
**M · depends on B-01 · build before the first SDK**

One executable test suite any SDK must pass: sync result, async callback, business error, infra error, retry with a stable idempotency key, trace propagation, deadline handling, schema validation, restart behaviour per A-03.

Written once, run against every language. Without it, three SDKs means three subtly different engines and support tickets you cannot reproduce.

### B-06 · SDK: TypeScript / Node
**M · depends on B-02, B-05**

Two entry points from one core: plain function registration for scripts and serverless, and NestJS decorators for the framework case. The decorator marks a handler, the SDK exposes the route, handles ack/async/callback and idempotency, and completes when the handler resolves — so a customer writes an ordinary async method and gets durable orchestration.

Ship with a working example that survives an engine restart mid-call. That example is the README.

### B-07 · SDK: PHP / Laravel
**M · depends on B-02, B-05**

Same contract, framework-idiomatic: service provider, route macro or attribute, queue integration for the async path since Laravel shops already run queues. This is the SDK that proves the protocol is not TypeScript-shaped, and Laravel is a large, underserved population for durable orchestration.

### B-08 · Worker registration lifecycle
**S–M · depends on B-01**

Register, health, deregister, drain. What happens when a worker is unreachable — how long a step waits, what the error taxonomy says, whether the run fails or parks. Also multiple workers behind one capability, and versioned capability rollout without breaking in-flight runs (interacts with A-05).

### B-09 · Guard sub-workflow recursion
**S**

Children spawn through the same `WorkflowSupervisorName` route as any run — no depth counter, no ancestry check, no cycle detection. A self-referencing schema is an unbounded fork bomb. Once agents compose workflows from data, this becomes triggerable **by model output**, not by a badly written schema. Depth counter in `SubWorkflowRef`, configurable max, ancestry cycle detection at spawn.

### B-10 · Pass input into sub-workflows
**S · depends on A-01**

`handleSubWorkflowAction` accepts and journals `input` and never passes it to the child. Wire it. Also add an output selector: the completion message carries the child's **entire** unfiltered per-node output map (`AggregatedOutputSnapshot()`), which balloons the parent's aggregated output and compounds A-09.

Both are prerequisites for an agent composing and running a sub-workflow with parameters.

---

## Tier C — Agents in Core

### C-01 · Agent node ↔ registry tool binding
**M · depends on B-03**

`ai/agent` and `ai/chat` exist as async functions. Bind them to registry capabilities so an agent's tool set is declared, versioned, and traced identically to a node call. An agent invoking a capability must be indistinguishable, in the journal and in the trace, from a graph node invoking it.

### C-02 · Memory as a first-class concept
**M · overlaps A-11(a) — do them together**

Working (run-scoped), episodic (past runs and conversations), semantic (durable facts). `ai/agent`'s `maxContextTokens`/`contextStrategy` trims inside one node execution and there is no graph-level equivalent, so conversation state grows unbounded in `aggregatedOutput` and in every journal payload.

Semantic memory in markdown: human-readable, diffable, reviewable, auditable. An enterprise buyer can read what the system believes; a vector blob cannot be audited. Writes are proposals by default, resolved by policy or by a human gate.

### C-03 · Model gateway
**M**

Provider-agnostic routing: OpenRouter as the default breadth play, Azure OpenAI / Bedrock / vLLM for self-hosted enterprise. Policy per node — `{primary, fallback[], maxCost, maxLatency}` — with budget enforcement and redaction at the boundary before anything reaches a telemetry sink.

Self-hosted models are a launch requirement for the regulated customers in the pipeline, not a roadmap item.

### C-04 · GenAI telemetry and cost
**M · depends on A-08, C-03**

Spans exist (`workflow.execute`, `node.execute`) with `workflow.*`/`node.*` attributes. Missing: agent id, model, tokens in/out, cost, gate decision, tenant. LLM usage is Prometheus-only (`ai/usage.go`), disconnected from the trace.

Follow OTel GenAI conventions where they exist; namespace the rest. Meter cost at the gateway, tagged run and step, so "what did this customer's Tuesday cost" is a query.

### C-05 · Human gate payload schema
**M · the Core half of the console's dynamic UI**

A gate is not a string. Define in Core the typed payload a gate carries — options (single/multi), confirm with summary, form with schema — and the correction semantics where rejecting one item of a multi-item confirmation re-opens **only that item** and preserves the rest without restarting the run.

Core owns the schema and the state transitions; Enterprise owns rendering (D-02). This split is what lets a customer build their own frontend against the same contract.

Depends on A-11 being resolved, since correction requires addressable per-turn state.

### C-06 · Durable timers and expiry sweeper
**M**

Sleeps, awakeable timeouts and workflow timeouts are per-actor `SendAfter` — lost on restart, with no sweeper (the only periodic actors are `workflow_claim_actor.go` and `cron_scheduler.go`). A 72-hour gate that outlives a deploy has no deadline. The `Awakeable` row already carries `DeadlineAt`. Periodic sweep for expired pending awakeables and sleeps, re-arm live timers on boot, claim-aware so two nodes do not fire the same timeout.

### C-07 · Expose awakeable IDs
**S**

Minted in-engine, written only into a journal entry's `Data` map. `GET /v1/workflows/{id}` returns `{workflowId, status}`; `SnapshotTimelineEvent` drops `Data`; snapshot and trace persist only at terminal state. `AwakeableRepository.FindPending(workflowID)` exists with no route in `mux_worker.go`. A caller cannot learn the ID it must call back with — a documented feature no external caller can use.

### C-08 · ForEach state survives restart, and stops silently lying
**M**

Iteration bookkeeping lives in `forEachStates`/`iterThreadToForEach` — plain Go maps, never reconstructed — so an in-flight ForEach does not survive a restart, even though the journal already writes four foreach entry types and reads none. Same class of bug as A-02.

Worse: `spawnForEachBatch` dispatches straight to the function pool, **bypassing system-function interception**, so `system/subworkflow`, `system/sleep` and `system/wait` inside a ForEach body hit the no-op placeholder, return success instantly, and silently do nothing. Silent wrong answers are worse than crashes. If the real fix is large, ship schema validation that **rejects** those functions inside a ForEach body in the meantime.

### C-09 · Mid-flight run read model
**M**

`GET /v1/workflows/{id}` returns `{workflowId, status}`. Snapshot and trace persist only in `sendWorkflowCompleted`. No SSE, no WebSocket; `memory_bus.go` is in-process and unreachable externally.

With everything inside the engine, this is no longer optional: the console has nothing to render without it. Minimum: `GET /v1/workflows/{id}/state` built from the journal — thread positions, completed steps, pending steps, open gates. Streaming is a separate later task; do not conflate them.

### C-10 · Per-node idempotency convention
**M · depends on B-01**

Today `idempotency/store.go` is trigger-level only. Nothing derives a key from `(runID, execID, attempt)`, while the resume path re-executes pending steps — at-least-once with no dedup beneath it. `ExecutionInfo` already carries `WorkflowID` and `ExecID` into every function, so the raw material is there. Derive a stable key, put it in the protocol envelope, and let the SDKs dedupe on it transparently.

### C-11 · Claim failure must fail closed
**S**

`claimForThisNode` returns `true` on a claim-store error and runs the workflow anyway. Under Postgres pressure that is a split-brain path with side effects. Fail closed, log, retry next sweep. Also revisit `ClaimWorkflows(nodeID, 10)` every 5s — recovering 50 runs after a node loss takes ~25 seconds.

### C-12 · Multi-thread resume correctness
**M · premise is inference · depends on F-02**

`buildResumeAction` walks `lastCompletedThreadIDs` and returns the first non-noop action, so a multi-thread run that crashed between thread completions may lose its other branches. No test covers it either way. Verify with the F-02 multi-thread scenario first; if it reproduces, resume must fan out to every pending thread.

---

## Tier D — Enterprise layer

Separate repo, separate release cadence, consumes Core's public API only. Starts once Tier A is green — building a console on an engine whose durability is unverified means debugging both at once.

- **D-01 · Tenancy, RBAC, SSO.** Row-level isolation enforced at the data layer. Approval rights distinct from execution rights.
- **D-02 · Console.** Schema editor, run inspector, gate approval rendering the C-05 payloads, trace rendered *as the workflow graph* so building and debugging share one mental model. This replaces the portal already started.
- **D-03 · Collaboration.** Multiple humans and agents on one run, server-authoritative sync, gate claiming so two people cannot approve the same thing twice, actor attribution on every action.
- **D-04 · Cost and usage dashboards.** Consumes C-04.
- **D-05 · Audit export.** Immutable log of runs, decisions, approvals, memory writes.
- **D-06 · Packaging.** Compose for pilot, Helm for production. Core's floor is already right — `CLUSTER_ENABLED=false`, `HA_ENABLED=false`, Postgres plus a writable path, embedded migrations, etcd and Redis optional. Do not regress it.

---

## Found during verification — unfiled

Raised by F-01 verification (2026-08-11), per "do not fix bugs found along the way". Not yet assigned
ids or tiers — that is the maintainer's call. Each says plainly whether it was **executed** or **read**.

### V-a · Agent tool calls are invisible to OTel *and* Prometheus
**Read, from control flow · relevant to C-04 and to B-03's parity requirement**

`internal/packages/agent_tools.go:86` (`InvokeTool`) calls `LoadedPackage.ExecuteFunctionSync`
(`internal/packages/loaded_package.go:50`), which goes straight to the transport and never enters
`internal/actors/workflow_func.go`. That file is the **only** creator of the `node.execute` span
(`:90`) and the only caller of `recordNodeDuration` (`:181`). So every tool an agent calls produces no
span and no `fuse_node_exec_duration_seconds` sample.

This is direct evidence on B-03's hard design constraint — "an agent calling a capability and a node
calling the same capability must produce the same journal entries, the same span shape and the same
idempotency behaviour". Today they demonstrably do not. Distinct from the known gap that intercepted
`system/*` functions have no node span.

### V-b · Resume appends a second `subworkflow:started` for one exec id, and nothing catches it
**Executed (`internal/actors/f01_2_verification_test.go`) · falls out of F-01.2, fix belongs with A-02**

After a resume the journal holds two `subworkflow:started` entries for a single exec id with *different*
`childWorkflowId` — `internal/actors/workflow_handler.go:917-926` appends unconditionally with no read
of an existing entry. Any future replay handler for `subworkflow:started` must therefore tolerate or
repair pre-existing duplicates rather than assume one entry per exec.

Postgres would not catch it either: `sub_workflow_refs` is UNIQUE on `child_workflow_id` only — there is
no unique index on the parent exec.

### V-c · `Workflow.Next` nil-derefs on a missing node, crashing the run
**Executed (`TestWorkflow_Next_AfterLiveSchemaEditDropsInFlightNode_MustNotPanic`)**

`internal/workflow/workflow.go:312` discards the error from `graph.FindNode` (`currentNode, _ := ...`)
and `:314` immediately dereferences the result, panicking in `internal/workflow/node.go:68` with
`runtime error: invalid memory address or nil pointer dereference`.

Any missing-node condition — a live schema edit, replay against a changed schema, a bad audit entry —
takes down the workflow actor instead of failing the run. A-05 (version pinning) and A-06 (deep copy)
each remove *a* way to reach it; neither makes `Next` safe. Cheap to fix independently.

---

## Dispatch order

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

**Sprint 1 — prove the engine.** F-01, F-02, F-03, A-07, A-06. Nothing user-visible ships; everything after becomes measurable.

**Sprint 2 — write the spec.** B-01, plus A-01, A-04, A-05 in parallel. The spec is a document and a decision, not code, and it gates the entire B tier — do not let it slip behind implementation.

**Sprint 3 — the pillar.** B-02, B-05, then B-06. One SDK, one conformance suite, one example that survives a restart mid-call.

**Gate — do not choose an agent execution model before F-02 is green.** The report says cycles are genuinely supported, sub-workflow spawn works, and awakeables persist. Two of those three already carry known restart defects. Choosing on the strength of prose is how you end up with an elegant model resting on a primitive that has never been executed under SIGKILL.

---

## Rules for the implementing agent

- **Verify before fixing.** Every task cites a report, not the code. If the premise does not hold, close it and report it.
- **One task, one PR.** They are deliberately separable.
- **A test reproducing the bug lands before the fix**, in the same PR, failing in the first commit.
- **Do not fix bugs found along the way.** Add them here with evidence and keep going.
- **The protocol spec is a contract.** Once B-01 is published and an SDK exists, changing it breaks strangers. Treat it with more care than the engine internals.
- **Core stays usable without Enterprise.** If a Core feature only makes sense with the commercial layer, it belongs in the commercial layer.
- **Public API changes need a changelog entry and a migration path.** Apache-2.0, with users who are not you.
