# FUSE core — implementation backlog

> **SUPERSEDED — do not work from this file.** Replaced by [`BACKLOG_V2.md`](BACKLOG_V2.md)
> ("FUSE — product shape and backlog"). This version was written for a separate TypeScript
> platform sitting on top of the engine; that framing is gone and **every task id was
> renumbered**. Citing an id from this file will send you to the wrong task. Kept only for
> provenance of the Phase 0 findings.

Derived from the Phase 0 capability report (`core/ @ 31c43e1`). Every task cites the evidence it came from.

**Read this first.** These tasks are written against a *report about* the code, not the code. Several findings are explicitly marked as inference in the report. **Task F-01 exists to verify before anything is built.** If a task's premise fails verification, close it and say so — do not implement around a bug that does not exist.

Tiers are dispatch order, not priority theatre:

- **F — Foundation.** Verification and the safety net. Nothing else starts until these pass.
- **A — Blocks Phase 3.** Without these, the ThinkAssist slice cannot run on FUSE at all.
- **B — Blocks the pilot.** The slice runs without them; a paying customer does not.
- **C — Post-pilot.** Real, scheduled, not now.

Sizes: **S** ≈ under a day, **M** ≈ a few days, **L** ≈ a week or more with design risk.

---

## Tier F — Foundation

### F-01 · Verify the three high-leverage findings
**Size S · no dependencies**

Three findings carry disproportionate weight and two of them are read from control flow rather than executed. Confirm each with a failing test before writing any fix.

1. `internal/actors/workflow_func.go` — the node span context is discarded (`_ = nodeCtx`), so function code cannot create child spans or inject `traceparent` outbound.
2. Restart with a pending `system/subworkflow` step spawns a **second child workflow** — `findPendingThreads` matches `step:started` without completion, `replayPendingThread` re-issues the action, interception fires again.
3. `internal/repositories/graph_memory.go:FindByID` returns the shared `m.graphs[id]` pointer that live `Workflow`s hold, so `graph_service.go:updateVersioned` → `graph.UpdateSchema` rewrites a running run's topology in place.

**Done when:** each is a checked-in test that fails on `main`, or a written note that the finding did not reproduce and why.

### F-02 · Crash-resume e2e harness
**Size M · depends on nothing · blocks everything in A**

The report is unambiguous: no test kills the process mid-run and asserts correct resume. `Workflow.Resume()` is called from one place (`workflow_handler.go:156`) and has no unit test. Everything Stellwerk sells rests on this path.

Build a harness that runs FUSE as a real process against real Postgres + fs object store, kills it with SIGKILL at a controllable point, restarts, and asserts final state. Not in-process — the existing `tests/e2e/workflow_resilience_suite_e2e_test.go` already covers in-process failure and that is not the same thing.

Scenarios, each its own test:
- kill after a completed step, mid-thread → resumes, no re-execution of the completed node
- kill with a pending async node → asserts exactly-once *or* documents at-least-once explicitly
- kill with a pending `system/subworkflow` → **exactly one child exists after restart** (this is F-01.2's regression test)
- kill with a pending `system/sleep` → remaining duration is honoured, not restarted
- kill with a pending awakeable → the token issued before the crash still resolves
- kill a multi-thread run between thread completions → all branches resume (covers the `buildResumeAction` inference)
- restart three times on the same run → journal length and rendered trace are stable

**Done when:** the harness runs in CI, and each scenario is either green or has a linked task in this backlog. Several will be red at first — that is the point. Do not fix them here.

### F-03 · Config validation for driver combinations
**Size S**

`DB_DRIVER=postgres` with the default `OBJECT_STORE_DRIVER=memory` produces durable journal rows in Postgres whose `input_ref`/`result_ref`/`data_ref` point into a store that dies with the process. `LoadAll` then fails on every payload fetch and no run is recoverable. Silent, and it looks correct until the first restart.

Extend `Config.Validate` (`internal/app/config/config.go`) to reject persistent-DB + ephemeral-object-store, and any other mismatched pair. Fail at boot with a message that names both variables.

**Done when:** boot fails fast on the mismatch; test covers each combination.

---

## Tier A — Blocks Phase 3

### A-01 · Input payload on workflow trigger
**Size S · high value, low risk**

`dtos.TriggerWorkflowRequest` is `{schemaID, idempotencyKey, environment}` — no payload. The plumbing is built and then dropped: `messaging.TriggerWorkflowMessage` has `Input`, `NewTriggerWorkflowWithInputMessage` exists, webhook/cron/event triggers populate it — but `workflow_sup.go:spawnWorkflowActor` takes three args, `WorkflowHandlerInitArgs` has three fields, and `workflow.go:Trigger` hardcodes `Args: map[string]any{}`.

Thread `Input` through: DTO → handler → `TriggerWorkflowMessage` → `spawnWorkflowActor` → `WorkflowHandlerInitArgs` → `Trigger(Args: input)`. Make it reachable from `SourceFlow` mappings under a reserved key (`trigger.*` or similar) so schemas can map trigger input into the first node.

**Done when:** `POST /v1/workflows` with a body reaches the first node's inputs; e2e asserts it; the same key works after a restart (it is journaled with the run).

**Why it blocks:** without it, every run starts by calling back into TypeScript to fetch its own input. One extra HTTP hop, one extra failure mode, on every single turn.

### A-02 · Generic async HTTP node
**Size M · the single most important new capability**

The transport layer is in-process only — `pkg/transport/type.go` declares HTTP and gRPC constants with no implementation, and `loaded_package.go:MapToRegistryPackage` registers non-internal functions metadata-only ("function %s has no transport"). The synchronous `fuse/pkg/http/request` blocks a pool worker for the call duration, defaults to a 10s timeout, and three concurrent long calls exhaust a run's 3-worker pool.

Meanwhile the async primitive already exists and no node uses it: `NewFunctionResultAsync()` plus `POST /v1/workflows/{workflowID}/execs/{execID}`.

Build `fuse/pkg/http/request_async`: POST to a configured URL, return `NewFunctionResultAsync()`, resume on callback. Requirements:
- the outbound request carries `workflowId`, `execId`, `nodeId`, `threadId`, and the callback URL in the body — **the execID is the callback token**, which sidesteps the awakeable discoverability problem (B-01) entirely
- `traceparent` injected into outbound headers (depends on A-05)
- per-node timeout with a durable deadline (depends on B-02) — for Phase 3 an in-memory timer is acceptable if the limitation is written down
- does not hold a pool worker while waiting
- callback validates that the exec is genuinely pending and rejects duplicates

**Done when:** a workflow with this node survives a 90-second TypeScript call, and survives a restart while the call is in flight (behaviour under restart is asserted, whichever behaviour you choose — see A-03).

### A-03 · Decide and enforce replay semantics for pending async nodes
**Size M · depends on A-02, F-02**

`findPendingThreads` matches `step:started` without completion — exactly the shape of an in-flight async node — and re-issues the action. So after a restart the TypeScript side gets called a second time with a **new execID**, and the original callback is orphaned.

Two options; pick one, write it in an ADR, enforce it in code:

- **(a) At-least-once, TS-side idempotent.** Cheap. The outbound call carries a stable `(workflowId, nodeId, attempt)` key; TypeScript dedupes; the old execID's callback is rejected as stale. Requires B-05 to give callers a stable key.
- **(b) Replay-aware.** Extend `replayJournalEntries` to recognise pending async steps and re-arm the callback with the *original* execID instead of re-dispatching. Correct, and the same machinery A-04 needs.

Given A-04 lands anyway, (b) is probably the honest answer — but (a) unblocks Phase 3 faster. Decide explicitly rather than by omission.

**Done when:** the corresponding F-02 scenario is green and the ADR states which guarantee FUSE offers.

### A-04 · Replay the intercepted system functions
**Size M · depends on F-02**

`replayJournalEntries` handles five of eighteen journal entry types: `thread:created`, `step:started`, `step:completed`, `thread:finished`, `state:changed`. `JournalAwakeableCreated`, `JournalSleepStarted` and `JournalSubWorkflowStarted` each have one write site and **zero read sites**. The rows are written and never read back.

Consequences, in descending severity:
- **subworkflow** — restart spawns a duplicate child. A real duplicate side effect.
- **awakeable** — restart re-mints the token via `uuid.New()`; the ID you handed the client is orphaned pending forever.
- **sleep** — restart restarts the full duration from zero.

Add replay handlers for all three. Minimum for Phase 3 is **subworkflow** (duplicate side effect) — awakeable and sleep can follow in B if the Phase 3 design uses A-02 for gates instead of awakeables.

**Done when:** the three F-02 scenarios are green.

### A-05 · Trace context end to end
**Size M**

The report calls TS → Go → TS three disconnected traces. Three separate breaks:

1. **Inbound.** `internal/handlers/trigger_workflow.go:HandlePost` never reads request headers; `startRootSpan` calls `tracer.Start(context.Background(), ...)`. The provider has `ExtractCarrier` and it is never called at the HTTP boundary. → extract `traceparent` from the trigger request and parent the root span to it.
2. **Handler → worker.** Already works (`InjectCarrier` on `ExecuteFunction`, `ExtractCarrier` in `workflow_func.go`). Leave alone, add a regression test.
3. **Outbound.** `_ = nodeCtx` in `workflow_func.go:96` discards the span context. `ExecutionInfo` has no `context.Context` field, so function code cannot create children or inject `traceparent`. → add the context to `ExecutionInfo` and use it in A-02.

Apply to every entry point that starts a run: HTTP trigger, webhook, event, cron (cron has no inbound parent — give it a documented root).

**Done when:** one trace spans TypeScript → FUSE root → node → outbound HTTP → TypeScript, verified against a real collector, not a unit test.

### A-06 · Pin runs to a schema version
**Size M**

`STELLWERK.md` §4 rule 2 requires published versions to be immutable and runs pinned. Today they are not: the `workflows` table has `schema_id` and **no version column**, and `postgres/workflow.go:Get → loadGraph(ctx, schemaID)` reads the *active* definition. A run recovered after an edit replays against whatever topology is current. Rollback is worse — it writes a new active version, so in-flight runs get their topology swapped silently, unrecorded in their own journal.

The retrieval half already exists: `FindByIDAndVersion` is implemented on the service and both repositories. The run simply never records where it started.

- migration: add `schema_version` to `workflows`
- record it at trigger time, journal it in the run's first entry
- `loadGraph` resolves by `(schemaID, version)`
- rollback and version-activate must not touch in-flight runs

**Done when:** a run started on v3 completes on v3 after v4 is published and activated mid-run; test asserts it across a restart.

### A-07 · Stop the shared-pointer live edit
**Size S · depends on F-01.3**

`graph_memory.go:FindByID` returns `m.graphs[id]` — the same pointer live `Workflow`s hold — so a `PUT /v1/schemas/{id}` rewrites a running run's nodes, edges and thread assignments mid-execution. The postgres driver rebuilds a fresh `Graph` per call and is safe in-memory.

Return a deep copy from the memory driver, matching postgres semantics. Assert both drivers behave identically.

**Note:** A-06 fixes the *durable* half of this (resume against the right version). A-07 fixes the *live* half. Both are needed; neither substitutes for the other.

**Done when:** editing a schema mid-run leaves the running instance's topology untouched under both drivers.

### A-08 · Fix duplicate journal append on replay
**Size S**

`replayJournalEntries` calls `SetResultFor`, and `SetResultFor` appends a journal entry. `Journal.LoadFrom` sets `lastPersisted = seq`, so replay-generated duplicates get fresh sequence numbers and are persisted by `persistWorkflowState()` at the end of `Init`. The journal grows by a full copy of its completed steps on every restart, and `trace_builder.go:BuildTrace` renders each step N+1 times after N restarts.

Suppress journaling during replay — a replay flag on the workflow, or a `SetResultForReplay` that skips the append. The former is less surface area.

**Done when:** journal length and rendered trace are byte-identical after three restarts (the F-02 stability scenario).

**Why it blocks:** it compounds. Each restart makes the next one heavier, and it corrupts the trace — which is the thing you are selling.

---

## Tier B — Blocks the pilot

### B-01 · Expose awakeable IDs
**Size S**

The token is minted inside the engine and written only into a journal entry's `Data` map. `GET /v1/workflows/{id}` returns `{workflowId, status}`; `SnapshotTimelineEvent` deliberately drops `Data`; snapshot and trace are persisted only at terminal state. `AwakeableRepository.FindPending(workflowID)` exists with no route in `mux_worker.go`.

Add `GET /v1/workflows/{id}/awakeables` (pending only) and include the ID in whatever B-03 exposes.

**Note:** if Phase 3 uses A-02 for gates, awakeables are not on the critical path — but they are a documented FUSE feature that is currently unusable by any external caller. As the maintainer of a public OSS engine, that is a bug regardless of what Stellwerk needs.

### B-02 · Durable timers and an expiry sweeper
**Size M**

Sleeps, awakeable timeouts and workflow timeouts are all per-actor `SendAfter`. They vanish on restart and there is no sweeper — the only periodic actors are `workflow_claim_actor.go` and `cron_scheduler.go`. A 72-hour human gate that outlives a deploy has no deadline at all.

The `Awakeable` row already carries `DeadlineAt`. Add a periodic sweeper that scans expired pending awakeables and expired sleeps, and re-arms in-memory timers for live ones on boot. HA-aware: sweeping must respect claims so two nodes do not fire the same timeout.

**Done when:** a gate with a 60s timeout, opened before a restart that takes 90s, fires its timeout after boot.

### B-03 · Mid-flight run read model
**Size M**

`GET /v1/workflows/{id}` returns `{workflowId, status}`. Snapshot and trace are persisted only in `sendWorkflowCompleted` — absent or stale for the entire life of a run. No SSE, no WebSocket; `internal/events/memory_bus.go` is in-process and unreachable from TypeScript.

Stellwerk's own design sidesteps this by keeping the read model in TypeScript, fed by push. But FUSE cannot be operated or debugged without it, and every other consumer of the engine has the same hole.

Minimum: `GET /v1/workflows/{id}/state` returning current thread positions, completed steps, pending steps and open gates, built from the journal. Streaming is a separate, later task — do not conflate them.

### B-04 · ForEach state survives restart
**Size M**

Iteration bookkeeping lives in `WorkflowHandler.forEachStates` / `iterThreadToForEach` — plain Go maps, never reconstructed from the journal — so a ForEach in flight does not survive a restart. The journal already writes `foreach:started`, `foreach:iteration:started`, `foreach:iteration:completed`, `foreach:completed` and never reads them: the same class of bug as A-04.

Also fix, or document loudly: `spawnForEachBatch` sends `RunFunctionAction` straight to the function pool, **bypassing system-function interception**. So `system/subworkflow`, `system/sleep` and `system/wait` inside a ForEach body hit the no-op placeholder, return success instantly, and silently do nothing. That is a silent-wrong-answer bug — worse than a crash. If the fix is large, ship schema validation that **rejects** those functions inside a ForEach body until it is fixed.

### B-05 · Per-node idempotency convention
**Size M**

`internal/idempotency/store.go` is trigger-level dedup only (`trigger_workflow.go`, `cron_scheduler.go`, `event_trigger.go`). Nothing derives a key from `(runId, execId, attempt)`, and the engine's own resume path re-executes pending steps — so at-least-once side effects are the documented behaviour with no dedup layer beneath it.

`ExecutionInfo` already carries `WorkflowID` and `ExecID` into every function, so the raw material is there. Add a derived, stable idempotency key to `ExecutionInfo`, make the built-in HTTP nodes send it as a header, and document the contract for custom functions. **Stable across replay** is the whole point — see the A-03 decision.

### B-06 · Distinguish retry attempts
**Size S**

Automatic retries reuse the same `ExecID` (`HandleNodeFailure` → `RetryFunctionAction{RunFunctionAction{FunctionExecID: execID}}`). Attempts are separable only via `step:retrying` entries and the in-memory `RetryTracker`, which is not rebuilt on replay. Manual retry does it correctly — `RetryNode` mints a fresh ExecID and records `previousExecId`.

Add an attempt counter to the journal entry and rebuild `RetryTracker` on replay, so "this node failed twice then succeeded" is answerable from durable state. Cost attribution and the trace view both need it.

### B-07 · Multi-thread resume correctness
**Size M · depends on F-02 · premise is inference**

`buildResumeAction` walks `lastCompletedThreadIDs` and returns the first non-noop action, so a multi-thread run that crashed between thread completions may lose its other branches. The report flags this as read from control flow with no test either way.

Verify with the F-02 multi-thread scenario first. If it reproduces, resume must fan out to every pending thread, not the first.

### B-08 · Claim failure must fail closed
**Size S**

`claimForThisNode` returns `true` on a claim-store error and runs the workflow anyway (there is a "running anyway" log line). Under Postgres pressure that is a split-brain path — two nodes executing the same run, with side effects.

Fail closed: on claim-store error, do not claim, log, retry on the next sweep. Also revisit `ClaimWorkflows(nodeID, 10)` every 5s — recovering 50 runs after a node loss takes ~25 seconds.

### B-09 · Payload write amplification
**Size M**

Every journal entry carrying a payload is one object-store PUT, with no size threshold, no inlining, no compression, no dedup. A 200 KB conversation transcript is re-PUT on every `step:started` whose input includes it and every `step:completed` whose result includes it, plus a third time into `output.json` on every `Save`. At 50 concurrent runs × ~12 steps that is ~600 PUTs and ~120 MB per round of turns. `LoadAll` then re-fetches the entire payload graph on every resume.

Two changes, independently valuable:
- **inline small payloads** in the PG row below a threshold, skipping the object store entirely
- **content-address large payloads** by hash, so an unchanged transcript is stored once and referenced N times

The second one alone removes most of the amplification for conversational workloads, which is exactly Stellwerk's shape.

### B-10 · Configurable pool size
**Size S**

`WorkflowFuncPool` is `PoolSize: 3` per workflow instance (`workflow_func_pool.go:Init`). Any blocking node caps a run at three concurrent branches, and one 90-second call holds a third of that run's capacity. Make it configurable globally and per-schema. A-02 reduces the pressure; it does not remove it.

### B-11 · Guard sub-workflow recursion
**Size S**

Children spawn through the same `WorkflowSupervisorName` route as any run — no depth counter, no ancestry check, no cycle detection. A self-referencing schema is an unbounded fork bomb. Since Stellwerk models sub-agents as sub-workflows, an agent that can pick its own sub-agent can trigger this from *data*, not from a bad schema.

Add a depth counter in `SubWorkflowRef`, a configurable max, and ancestry cycle detection at spawn.

### B-12 · Pass input into sub-workflows
**Size S · depends on A-01**

`handleSubWorkflowAction` accepts and journals the `input` parameter and then never passes it to the child. Once A-01 exists, wire it: the child's `Trigger` args come from the parent's declared input mapping.

Also: the completion message carries the child's **entire** per-node output map, unfiltered (`AggregatedOutputSnapshot()`). Add an output selector so a parent can take one path instead of the whole child state — otherwise nested runs balloon the parent's aggregated output, compounding B-09.

---

## Tier C — Post-pilot

### C-01 · Address node output by execution
**Size L · the only task here that changes the data model**

`SetResultFor` writes `w.aggregatedOutput.Set(entry.FunctionNodeID, ...)` — the KV is keyed by **node ID, not exec ID**. A node that runs twice overwrites its own previous output, and `SourceFlow` only ever sees the latest execution. The journal keeps every execution; the data plane does not.

This is the structural mismatch with `STELLWERK.md` §5.2: per-step history is not addressable from inside the graph, so a "reject step 2, keep steps 1, 3, 4" correction cannot be expressed in FUSE's own data plane.

**Workaround for the pilot:** keep selection history in the TypeScript read model and pass the corrected set back in as fresh input. It works, and it means the *engine* is not the system of record for a correction — which is worth writing down as a known limitation rather than discovering in front of a customer.

Real fix: key by `(nodeID, execID)` with a `latest` alias for existing schemas, and a mapping syntax for addressing a specific execution. Breaking-ish; needs its own ADR and a migration story.

### C-02 · Runtime graph mutation
**Size L · only if Phase 1 concludes it is needed**

Today: no. `NewGraph` computes once from `GraphSchema`; `Workflow` holds that pointer and never re-derives; there is no add-node/add-edge on `Graph`, no such workflow action, no such message. The existing dynamism is conditional routing over declared edges, cycles (genuinely supported by `calculateThreads`), ForEach dynamic threads (width only, over already-declared nodes, linear body), and sub-workflow spawn.

If Phase 1 lands on the static supervisor loop plus sub-workflows, **close this task**. Do not build it speculatively. Revisit only when a real workflow cannot be expressed.

### C-03 · Out-of-process function transport
**Size L**

Make the HTTP and gRPC constants in `pkg/transport/type.go` real, so a node's implementation can be a service in another language and `MapToRegistryPackage` stops registering them metadata-only. A-02 covers Stellwerk's need with a single built-in node; this is the general version, and it is what makes FUSE interesting to people who do not write Go.

### C-04 · OTel GenAI attributes and cost
**Size M**

Spans exist (`workflow.execute`, `node.execute`) with `workflow.*` / `node.*` attributes. None of the `STELLWERK.md` §10 set exist — `.agent.id`, `.model`, `.tokens.in/out`, `.cost.usd`, `.gate.decision`, `.tenant.id` — and LLM usage is Prometheus metrics only (`ai/usage.go`), disconnected from the trace.

Follow OTel GenAI semantic conventions where they exist. **Design note:** most of these attributes belong to Stellwerk's gateway, not to FUSE. Decide the split before implementing, or you will emit them twice from two places and reconcile them forever.

### C-05 · Graph-level context management
**Size M**

`ai/agent`'s `maxContextTokens` / `contextStrategy` trims a transcript *inside one node execution*. There is no equivalent at graph level, so a conversation growing across steps grows unbounded in `aggregatedOutput` and in every journal payload. Partly mitigated by B-09; the policy question is separate from the storage question.

---

## Dispatch order

```
F-01 ─┬─ F-02 ──┬── A-03, A-04, B-07   (need the harness to define "correct")
      └─ A-07   │
F-03 ────────────┤
                 │
A-01 ─┬─ A-02 ───┘         A-01 also unblocks B-12
      └─ A-05 ── A-02 (outbound traceparent)
A-06 ── independent
A-08 ── independent, do early (compounds every restart)
```

**Suggested first sprint:** F-01, F-02, F-03, A-08, A-07. Verification and the safety net, plus the two cheap correctness fixes. Nothing user-visible ships, and every subsequent task becomes measurable.

**Second sprint:** A-01, A-05, A-02, then the A-03 decision. That is the Phase 3 slice.

---

## Rules for the implementing agent

- **Verify before fixing.** Every task cites a report, not the code. If the premise does not hold, close the task and report it.
- **One task, one PR.** These are deliberately separable. Do not bundle A-02 with A-05 because they touch adjacent lines.
- **A test that reproduces the bug lands before the fix**, in the same PR, failing in the first commit.
- **Do not fix bugs discovered along the way.** Add them to this backlog with evidence and keep going. Scope creep in an engine rewrite is how three-week projects become three-month ones.
- **Tier C is not a parking lot for hard problems.** If a Tier A task turns out to require C-01, stop and escalate rather than quietly starting it.
- **Public API changes need a note in the changelog and a migration path.** FUSE is Apache-2.0 with users who are not you.
