---
name: durable-execution-internals
description: Maps the engine's durability machinery — the 18 journal entry types and which are actually replayed, the resume control flow from process start to a running run, the invariants a change must preserve, and the known defects with their backlog owners. Use when touching journal, replay, resume, persistence or payload externalization, or before starting any F/A-tier backlog task that claims a durability bug.
---

# durable-execution-internals

The engine's durability story is "append-only journal + replay on recovery" (ADR-0010). The
journal defines **18** entry types. `Workflow.replayJournalEntries` handles **5**. That gap is the
single most important fact in this pack: most of what the journal records is written and never
read back, so several primitives silently do the wrong thing after a restart. Do not read a
write-only entry type as evidence that its behaviour is durable.

## When to use

- Changing anything under `internal/workflow/` (journal, replay, resume, threads, audit log,
  projections) or `internal/actors/workflow_handler.go`.
- Changing `internal/repositories/postgres/journal.go` or `.../workflow.go`.
- Adding a journal entry type, or changing what an existing one carries.
- Starting F-01, F-02, A-02, A-03, A-05, A-07, A-08, A-09, A-11, C-06, C-08, C-12 from the
  repo-root backlog (`BACKLOG_V2.md`). *If the maintainer has renamed it to `BACKLOG.md`, the
  canonical file is the one titled "FUSE — product shape and backlog".*

## Ground truth

Read these before trusting any prose, including this file.

| File | What it owns |
| --- | --- |
| `internal/workflow/journal.go` | The 18 type constants, `JournalEntry`, `Journal` (`Append`/`LoadFrom`/`NewEntries`/`MarkPersisted`). 136 lines, **zero read logic**. |
| `internal/workflow/workflow.go` | `Trigger`, `Resume`, `replayJournalEntries`, `buildResumeAction`, `findPendingThreads`, `replayPendingThread`, `Next`, `SetResultFor`, `HandleNodeFailure`, `RetryNode`. |
| `internal/actors/workflow_handler.go` | The actor owning one run: `Init` (trigger-vs-resume fork), system-function interception, all persistence calls. |
| `internal/repositories/postgres/journal.go` | Payload externalization (`input_ref`/`result_ref`/`data_ref`) and rehydration. |
| `internal/repositories/postgres/workflow.go` | The workflow envelope, `loadGraph`, `restoreState`. |
| `internal/repositories/postgres/migrations/000001_create_tables.up.sql` | The `journal_entry_type` ENUM and `journal_entries` schema. |
| `docs/adr/0010-durable-execution-journal-and-replay.md` | The intended contract. Useful as a spec to test against — **not** as a description of current behaviour (see Traps). |

## The two-writer rule

Journal writes are split across exactly two owners and the split is deliberate:

- **The aggregate** (`internal/workflow/workflow.go` + `foreach.go`) writes the execution-graph
  types via `w.journal.Append(...)`.
- **The actor** (`internal/actors/workflow_handler.go`) writes the system-primitive types by
  reaching in via `a.workflow.Journal().Append(...)`.

Keep new entry types on the correct side of that line. Aggregate state goes in the aggregate.
One exception already exists and is worth knowing about: `foreach:iteration:started` is a
system-primitive type written from the aggregate (`foreach.go:45`) because thread allocation
happens there — the other three `foreach:*` types are written by the actor.

## Journal entry inventory

`written` = the `Type:` field line of the `Append` call. `read` = a `case`/comparison that
structurally consumes the entry. Sites verified by grep in both directions.

| Entry type | Written at | Read / replayed at | Meaning |
| --- | --- | --- | --- |
| `step:started` | `workflow.go:171` (Trigger), `:488` (RetryNode), `:616` (newRunFunctionAction), `foreach.go:51` | `workflow.go:207` (replay → `auditLog.NewEntry`), `:273` (findPendingThreads), `trace_builder.go:22`, `execution_snapshot_builder.go:55` | A node execution was dispatched. Carries `Input`. The **only** thing that creates a trace step row. |
| `step:completed` | `workflow.go:430-434` (`SetResultFor`, `Output.Status == FunctionSuccess`) | `workflow.go:209` (replay → `SetResultFor`), `:280`, `trace_builder.go:35`, `execution_snapshot_builder.go:79` | A node returned success. Carries `Result`. Replaying it is what rebuilds `aggregatedOutput`. |
| `step:failed` | `workflow.go:430-434` (`SetResultFor`, status != success) | `workflow.go:280` (findPendingThreads), `trace_builder.go:47`, `execution_snapshot_builder.go:90`, `internal/repositories/journal_memory.go:53` + `postgres/journal.go:252` (`FindFailed`) | A node returned an error. **Not replayed** — a failed step's `Result` is never restored into the audit log on resume. |
| `step:retrying` | `workflow.go:857` (`HandleNodeFailure`) | `trace_builder.go:62`, `execution_snapshot_builder.go:104` | An automatic retry was scheduled. **Not replayed** — which is exactly why `RetryTracker` is not rebuilt (**A-08**). |
| `step:manual-retry` | `workflow.go:478` (`RetryNode`), `Data.previousExecId` | **NEVER READ** in non-test code | Links a manual-retry exec to the exec it replaces. `SnapshotNodeRun.PreviousExecID` exists and is never populated. |
| `thread:created` | `workflow.go:166` (Trigger), `:603` (cross-thread `newRunFunctionAction`) | `workflow.go:204` (replay → `threads.New`), `execution_snapshot_builder.go:42` | A thread entered the registry. **The only way replay learns a thread exists.** |
| `thread:finished` | `workflow.go:318`, `:327`, `:348`, `:358`, `:371`, `:884` | `workflow.go:211` (replay), `execution_snapshot_builder.go:50` | A thread reached its terminal node. Replay collects these into `lastCompletedThreadIDs`. |
| `state:changed` | `workflow.go:550` (`SetState`) | `workflow.go:217` (replay, **direct field write**, bypassing `SetState`), `trace_builder.go:68`, `execution_snapshot_builder.go:112` | Run-level state transition. |
| `sleep:started` | `workflow_handler.go:752`, `Data{duration, reason}` | **NEVER READ — owned by A-02** | A `system/sleep` gate opened. |
| `sleep:completed` | `workflow_handler.go:815` | **NEVER READ — owned by A-02** | The sleep timer fired. |
| `awakeable:created` | `workflow_handler.go:782`, `Data{awakeableId, timeout}` | **NEVER READ — owned by A-02** | An external-callback gate opened. |
| `awakeable:resolved` | `workflow_handler.go:849` | **NEVER READ — owned by A-02** | An external caller resolved the gate. |
| `subworkflow:started` | `workflow_handler.go:918`, `Data{childWorkflowId, childSchemaId, async}` | **NEVER READ — owned by A-02** | A child run was spawned. |
| `subworkflow:completed` | `workflow_handler.go:1186` | **NEVER READ — owned by A-02** | A child run reported back. |
| `foreach:started` | `workflow_handler.go:991` | **NEVER READ — owned by C-08** | A `system/foreach` began; `Data` carries the batch counts. |
| `foreach:iteration:started` | `foreach.go:45` | **NEVER READ — owned by C-08** | One iteration thread was allocated. **Note: no `thread:created` is written alongside it.** |
| `foreach:iteration:completed` | `workflow_handler.go:1071` | **NEVER READ — owned by C-08** | One iteration finished. |
| `foreach:completed` | `workflow_handler.go:1082` | **NEVER READ — owned by C-08** | All batches finished. |

**Partial-read caveat.** The ten "NEVER READ" types (and `step:manual-retry`, which is likewise
consumed by nothing) *are* appended to `ExecutionSnapshot.Timeline` by
`execution_snapshot_builder.go:32-39`, which records Sequence/Timestamp/Type/ThreadID/ExecID/NodeID
and **drops `Data`**. They are visible as opaque timeline events; nothing structural consumes them.

**Postgres ENUM gap.** The schema half is confirmed: `journal_entry_type` has 13 values from
migration `000001` plus `step:manual-retry` from `000004` = **14**, no migration anywhere adds a
`foreach:*` value, the Go file defines 18, and `postgres/journal.go:84` inserts `string(entry.Type)`
straight into that column. The runtime consequence below is **read off the code, not executed
against a live Postgres — verify it before acting on it**: under `DB_DRIVER=postgres` the first
`persistJournal` after a `system/foreach` node starts should fail the whole transaction, and since
`persistJournal` skips `MarkPersisted` on error (`workflow_handler.go:467-471`) the same poisoned
batch is retried and fails forever, freezing that run's journal at the last pre-foreach sequence.
Any C-08 work starts with an `ALTER TYPE ... ADD VALUE` migration (pattern:
`migrations/000004_add_manual_retry_journal_type.up.sql`) or it cannot be observed at all.

## Resume control flow, hop by hop

**Non-HA boot recovery** (the only recovery path when `HA_ENABLED=false` — it is a one-shot, there
is no periodic retry):

1. `internal/app/fuse.go:242` `Fuse.Start` → `node.Send` to `workflow_sup` with
   `{Type: RecoverWorkflows}` (`fuse.go:248-249`; `WorkflowSupervisorName == "workflow_sup"`).
2. `internal/actors/workflow_sup.go:212` `recoverWorkflows` → `FindByState(untriggered, running, sleeping)` (`:215`).
3. Per row: `workflowRepository.Get(id)` → `spawnWorkflowActor(schemaID, wf.ID(), wf.Environment())` (called at `workflow_sup.go:233`, declared at `:239`) → `StartChild(WorkflowInstanceSupervisor, ...)`.

**HA recovery**: `internal/actors/workflow_claim_actor.go:108` `sweep` →
`ClaimWorkflows(nodeID, 10)` (`postgres/claim.go:24`, `FOR UPDATE SKIP LOCKED`, states
untriggered/running/sleeping, steals only when `claimed_at < NOW() - HA_LEASE_TIMEOUT`) → one
`TriggerWorkflow` message per claimed row → same `spawnWorkflowActor`. The actor is only registered
when `HA_ENABLED=true` (`fuse.go:217`), and the only claim repository that does anything is the
Postgres one, selected only when `DB_DRIVER=postgres` **and** a pool exists
(`internal/app/di/repos.go:82-89`; otherwise `NewMemoryClaimRepository()`, logged as "no-op").
So in practice this path needs both.

**The resume path itself** — `internal/actors/workflow_handler.go:114` `Init`:

| Hop | Line | What happens |
| --- | --- | --- |
| 1 | `:129` | `workflowRepository.Exists` → true takes the resume branch. |
| 2 | `:130` | `workflowRepository.Get`. **Postgres** (`postgres/workflow.go:45`) builds a *fresh* `workflow.New` with empty threads/auditLog/aggregatedOutput, resolving the graph via `loadGraph(ctx, schemaID)` — the **active** definition, no version predicate (**A-05**). **Memory** (`internal/repositories/workflow_memory.go:39`) returns the *live pointer*. |
| 3 | `:133` | `claimForThisNode` (`:366`) — on loss it sends `NewWorkflowCompletedMessage(wfID, "claimed-elsewhere")` to `a.Parent()` and returns `false`, leaving persisted state untouched. Returns `true` on a claim-store **error** — "fails open", stated in its own doc comment at `:360-365` (**C-11**). |
| 4 | `:138-139` | `a.workflow.SetSecretResolver(a.newSecretResolver(a.workflow.Environment()))`, `graphService.EnsureNodeMetadata`. |
| 5 | `:143` | `startRootSpan`. Note `startWorkflowTimeout` is **not** called here — it is only called at `:193` on the fresh-create path, so a schema's total timeout is lost after any restart. |
| 6 | `:145` | `State() == StateUntriggered` ⇒ `Trigger()` instead of replay. |
| 7 | `:150` | `journalRepo.LoadAll`. On error it **logs and continues without `LoadFrom`** — see Traps. |
| 8 | `:154` | `Journal().LoadFrom(entries)` — sets `seq` to the last entry's `Sequence` and `lastPersisted = seq`. |
| 9 | `:156` | `workflow.Resume()`. |
| 10 | `:159` | `handleWorkflowAction(action)`. |
| 11 | `:163` | `persistWorkflowState()` = `persistJournal()` + `workflowRepository.Save`. **This is what flushes replay's duplicate appends (A-07).** |

**Inside `Resume()`** (`workflow.go:187`):

- Empty journal ⇒ `NoopAction`. Otherwise `replayJournalEntries` then `buildResumeAction`.
- `replayJournalEntries` (`:199-223`) iterates in sequence order and handles five types:
  `thread:created` → `threads.New` (which **resets** the thread to running, `thread.go:47`);
  `step:started` → `auditLog.NewEntry`; `step:completed` → **`SetResultFor`**, which sets the audit
  result, does `aggregatedOutput.Set(nodeID, ...)` **and appends a new journal entry**;
  `thread:finished` → `threads.Get(...).SetState(ThreadFinished)` + collect the id;
  `state:changed` → direct write to `w.state.currentState`. The other 13 fall through.
- `buildResumeAction` (`:226-257`): `findPendingThreads` builds `started[execID]` from
  `step:started` and `completed[execID]` from `step:completed|step:failed`, returning every
  started-but-not-completed exec.
  - **0 pending** ⇒ loop `lastCompletedThreadIDs` calling `Next(threadID)` and return the **first**
    non-noop action, discarding the rest (**C-12**).
  - **1 pending** ⇒ `replayPendingThread`.
  - **>1 pending** ⇒ `RunParallelFunctionsAction` over all of them (this branch fans out correctly).
- `replayPendingThread` (`:295-306`) returns `RunFunctionAction{FunctionExecID: workflow.ExecID(pt.execID)}`
  — the **original** execID from the journal, reused, and **no journal entry is appended**.

**Re-dispatch.** The action goes back through `handleWorkflowAction` → `handleWorkflowRunFunctionAction`
(`workflow_handler.go:671`), whose switch at `:675-688` re-intercepts `system/sleep`,
`system/wait`, `system/subworkflow`, `system/foreach` — which is precisely how A-02's duplicate
side effects happen.

## Invariants any change must preserve

1. **Sequence numbers are assigned solely by `Journal.Append`** (`journal.go:81`), which also
   stamps `entry.Timestamp = time.Now()` unconditionally. There is no way to append with a
   caller-supplied sequence or timestamp. Any replay-time append therefore carries a *replay-time*
   timestamp — plan around it, do not fight it in the projections.
2. **`lastPersisted` is a watermark, not content dedup.** `persistJournal` flushes `NewEntries()`
   (everything with `Sequence > lastPersisted`) and only then calls `MarkPersisted`; it
   deliberately does **not** mark on error (`workflow_handler.go:467-471`). One poisoned entry
   blocks every subsequent entry for that run forever.
3. **Postgres enforces `UNIQUE (workflow_id, sequence)`** (`idx_journal_wf_seq`). The memory
   driver does not dedup at all. A replay bug that re-appends an existing sequence errors loudly
   under Postgres and passes silently under memory.
4. **Replayed steps must be idempotent.** `replayPendingThread` re-issues the original action with
   no marker, so a re-dispatched node runs again with the same execID. Nothing beneath it dedupes
   (**C-10**).
5. **Payload refs must resolve.** `postgres/journal.go:160-206` fetches every non-nil
   `input_ref`/`result_ref`/`data_ref` through an errgroup with `SetLimit(10)`; a single failed
   object fetch aborts the whole `LoadAll`. Keys are deterministic and sequence-scoped:
   `workflows/{workflowID}/journal/{sequence}/{input|result|data}.json` (`journal.go:28`).
6. **Every entry type string must exist in both the Go constants and the Postgres ENUM.** Adding a
   Go constant without a migration is a runtime insert failure, not a compile error. Enum
   migrations are effectively irreversible (`000004`'s down file is a comment explaining why).
7. **Reconstruction must not journal.** `postgres/workflow.go:380` `restoreState` calls `SetState`
   (which appends `state:changed` at sequence 1) and relies on `LoadFrom` to discard it. A second
   reconstruction path must honour the same contract.
8. **Projections are pure.** `BuildTrace` and `BuildExecutionSnapshot` both say so in their doc
   comments. New read logic takes `[]JournalEntry` and returns a value. The one place this is
   violated today is `replayJournalEntries` calling the mutator `SetResultFor`.

## Known defects — do not mistake these for the contract

| Backlog id | Defect | Evidence |
| --- | --- | --- |
| **A-02** | Restart with a pending `system/subworkflow` mints a **second child** — `findPendingThreads` matches the `step:started` (`subworkflow:started` is never read), `replayPendingThread` re-issues, the switch re-intercepts, and `handleSubWorkflowAction` calls `workflow.NewID()` at `workflow_handler.go:902`. The duplicate child is what the backlog asserts. The follow-on — the first child row still being `running`/`sleeping` so recovery re-drives it too, and both children notifying the same `ParentExecID` and double-advancing the parent — is **inference from the control flow, not an observed run**; F-02 scenario 3 is what confirms it. | `workflow.go:267-306`, `workflow_handler.go:682-684`, `:901-933` |
| **A-02** | Restart with a pending `system/wait` calls `uuid.New()` again (`workflow_handler.go:728`), saving an **additional** awakeable row and appending an additional `awakeable:created`, and re-arms the timeout with the **full** original duration (`:789-794`). Both rows carry the same `ExecID`, so resolving either advances the run and resolving both double-advances it. *Note: the pre-crash token still resolves correctly, because `resolve_awakeable.go` looks up the awakeable's own stored `ExecID`/`ThreadID` and `replayPendingThread` reuses the original execID.* | `workflow_handler.go:727-747`, `internal/handlers/resolve_awakeable.go` |
| **A-02** | Restart with a pending `system/sleep` re-parses `Args["duration"]` from the journaled input and `SendAfter`s the **full** duration (`workflow_handler.go:710-763`). A 24h sleep with 10 minutes left restarts at 24h. The remainder is computable — `sleep:started`'s `Timestamp` and `Data["duration"]` are both persisted. | `workflow_handler.go:710-763` |
| **A-07** | **Duplicate journal append on replay — compounds every restart.** `LoadFrom` sets `lastPersisted = seq`; replay's `case JournalStepCompleted` calls `SetResultFor`, which unconditionally appends at `workflow.go:434` with a *fresh* sequence; `Init`'s `persistWorkflowState` (`:163`) flushes them. Each restart adds one duplicate `step:completed` per previously-completed step. **Constraint for the fix:** `SetResultFor`'s `aggregatedOutput.Set` (`:428`) is the only thing that rebuilds the data plane during replay — gate the `Append` at `:434` only; skipping the call empties aggregated output. | `journal.go:100-108`, `workflow.go:209`, `:428-440` |
| **A-07** | Trace corruption is real but not "each step N+1 times". `BuildTrace` only creates a step row on `step:started` (`trace_builder.go:22`), which replay does **not** duplicate. A duplicate `step:completed` mutates the existing row in place: `CompletedAt`/`Duration` are recomputed against the *replay* timestamp (`trace_builder.go:37-41`), so after a restart six hours later every completed step reports a ~6h duration, and `Status` is forced back to `completed` even for steps that ended `retrying`/`failed`. The snapshot **timeline** does grow by a full copy per restart (`execution_snapshot_builder.go:32`). | as cited |
| **A-09** | **Payload write amplification.** `postgres/journal.go:46-68` does one unconditional `putJSON` per non-nil `Input`, `Result` and `Data` — no threshold, no inlining, no compression, no dedup — and keys are sequence-scoped, so an identical payload at two sequences is stored twice. `postgres/workflow.go:81-97` re-PUTs the entire aggregated output to `workflows/{id}/output.json` on **every** `Save`, which `persistWorkflowState` calls on every state transition (`workflow_handler.go:392-397` = `persistJournal` + `Save`). `postgres/trace.go:75,83` adds two more PUTs per step per save. `LoadAll` re-fetches the whole *payload* graph — every `input_ref`/`result_ref`/`data_ref` object — on every resume. | as cited |
| **A-11** | **`aggregatedOutput` is keyed by node ID, not exec ID** — `workflow.go:428` is `w.aggregatedOutput.Set(entry.FunctionNodeID, result.Output.Data)`. A node that runs twice destroys its own history in the data plane; the journal keeps every execution, `SourceFlow` only ever sees the latest (read side: `applyFlowMapping`'s `aggregatedOutput.Get(mapping.Variable)`). This is the promoted Tier-A design risk. **Do not start by re-keying the KV** — the backlog wants an ADR first, and note `pkg/store/kv.go` is objx-backed with dot-notation, so a composite key needs a separator that is not `.`, and `SetResultFor` is also the replay rebuild path so any re-key changes replay semantics at the same time. | `workflow.go:428`, `pkg/store/kv.go:71-84` |
| **C-08** | **ForEach state does not survive restart.** `forEachStates map[string]*ForEachState` and `iterThreadToForEach map[uint16]string` (`workflow_handler.go:97-100`) are re-initialised to empty maps in `Init` (`:118-119`) and never reconstructed, although all four `foreach:*` types are written. Worse: `StartForEachIteration` (`foreach.go:42-51`) creates the dynamic thread with `threads.New` but journals `foreach:iteration:started`, **not `thread:created`** — so on resume the iteration thread does not exist in `w.threads` while its `step:started` still yields a pending thread; `Next` then does `w.threads.Get(threadID)` followed immediately by `currentThread.CurrentExecID()` with no nil check (`workflow.go:310-311`) — a nil-pointer panic. | as cited |
| **C-08** | **Interception bypass.** `spawnForEachBatch` (`workflow_handler.go:1043-1046`) sends the `RunFunctionAction` straight to `WorkflowFuncPoolName`, so the switch at `:675-688` never runs and `system/*` inside a ForEach body reaches the pool as an ordinary function (the implementations in `internal/packages/functions/system/` are success-returning placeholders). **A second, unlisted instance of the same bypass:** the `ActionRetryFunction` branch at `workflow_handler.go:658-668` also `SendAfter`s directly to the pool. | as cited |
| **C-12** | **Multi-thread resume — premise is inference, verify before fixing.** The "first non-noop wins" loss is real but only on the **zero-pending** branch (`workflow.go:229-238`); the pending branch fans out correctly. The reproducing shape is "crashed after several threads recorded `thread:finished` with no step in flight". Be warned: `Next()` has side effects — the discarded calls have **already** journaled their `step:started`, so the next restart picks them up as pending. Expect it to look flaky across one restart and self-heal across two. | `workflow.go:226-257` |

## Traps — looks right, is wrong

- **An async-parked step is journaled as `step:completed`, not left pending.**
  `NewFunctionResultAsync()` sets `Output.Status = FunctionSuccess` (`pkg/workflow/fn_result.go:43-49`),
  and `handleMsgFunctionResult` calls `SetResultFor` **before** checking `Result.Async`
  (`workflow_handler.go:275-281`). So `findPendingThreads` does **not** see an in-flight async node
  as pending, and there is no journal marker distinguishing "awaiting a callback" from "done".
  A-03 is the open decision on this: (a) at-least-once with a stable idempotency key the SDK
  dedupes on, or (b) replay-aware re-arming of the original execID. The backlog leans toward (a)
  ("the SDK contract makes (a) far more palatable"). *This pack's reading, not the backlog's text:*
  (b) would need a new "parked" entry type to tell the two states apart, and therefore an enum
  migration. Do not assume the pending-step machinery covers async.
- **`LoadAll` failure is silent and then corrupting.** `Init:150-155` logs and proceeds *without*
  `LoadFrom`. The journal still holds the sequence-1 `state:changed` that `restoreState` appended,
  `lastPersisted` is 0, `Resume()` sees a non-empty journal and returns `Noop`, and the trailing
  `persistWorkflowState` tries to insert sequence 1 again — violating `idx_journal_wf_seq`. This is
  exactly the F-03 combination (postgres journal + memory object store), and
  `internal/app/di/objectstore.go:38` falls through to memory for **any** unrecognized driver string.
- **Never validate a resume fix against the memory driver.** `MemoryWorkflowRepository.Get` returns
  the live `*Workflow` with populated threads/auditLog/aggregatedOutput; Postgres returns a fresh
  empty one. Replay under memory stomps live thread state (`threads.New` replaces the thread with a
  fresh running one) and re-registers audit entries on top of an already-populated aggregate. A
  green memory test proves nothing about Postgres.
- **`threads.AllFinished()` returns `true` for an empty thread map** (`thread.go:84-95`). A resumed
  run whose threads were not reconstructed can be declared complete by `checkWorkflowCompletion`
  while work is outstanding.
- **Replay never journals its own re-dispatch.** After N restarts a node that ran N+1 times still
  shows exactly one `step:started`. Per-attempt accounting (A-08, cost attribution) cannot be built
  from the journal as it stands.
- **ExecIDs are UUIDv8 with the thread id embedded** (`pkg/workflow/exec_id.go:13,23`) and only 12
  bits are decoded (low nibble of byte 6 + byte 7), capping thread ids at 4095. Async callbacks
  recover their thread purely by parsing the execID with no lookup. Never mint an execID with a
  plain `uuid.New()` for anything that returns through `handleMsgAsyncFunctionResult`.
- **A node ID containing `.` breaks the data plane silently.** `aggregatedOutput` is objx-backed
  with dot-notation paths, so `Set("a.b", …)` writes a nested path, not a flat key.
- **ADR-0010 overstates current behaviour.** Its L34-37 enumerates all 18 types as "covering the
  full lifecycle" and L40-43 says `Resume()` "replays entries to rebuild threads, audit log, and
  aggregated output" — true only for the five handled types. If A-02 lands, ADR-0010 needs an
  amendment or a superseding ADR, not a silent code change.
- **`GET /v1/workflows/{workflowID}/status` does not exist**, whatever older prose says.
  The route table (`internal/actors/mux_worker.go`) registers, for a run,
  `/v1/workflows/{workflowID}` (`:220`, returns `{workflowId, status}` only — `get_workflow.go:49-64`),
  `/cancel` (`:230`), `/snapshot` (`:240`), `/retry-node` (`:250`), `/retry` (`:260`),
  `/v1/awakeables/{awakeableID}/resolve` (`:270`), `/trace` (`:280`) and
  `/v1/workflows/{workflowID}/execs/{execID}` (`:79`). Trust `mux_worker.go`.

## Inspecting a run's durable state locally

```bash
make infra-up          # docker compose --profile infra up -d  (PG 5432, rustfs S3 9000, etcd 2379)
export DB_POSTGRES_DSN='postgres://fuse:fuse@localhost:5432/fuse?sslmode=disable'
make migrate           # builds bin/fuse, then ./bin/fuse migrate
make run               # ./bin/fuse server -o -p 9090 -l debug  (set DB_DRIVER/OBJECT_STORE_DRIVER first;
                       #  note -o also starts the ergo observer app — drop it for a plain engine)
```

Relevant tables (all created by the embedded migrations):
`workflows` (`workflow_id, schema_id, state, output_ref, snapshot_ref, environment, claimed_by, claimed_at`
— **no version column**, A-05), `journal_entries`, `awakeables`, `sub_workflow_refs`,
`execution_traces`, `execution_trace_steps`, `graph_schemas`, `graph_schema_versions`,
`node_heartbeats`, `idempotency_keys`.

```bash
psql "$DB_POSTGRES_DSN" -c "SELECT sequence, entry_type, thread_id, exec_id, function_node_id
  FROM journal_entries WHERE workflow_id = '<id>' ORDER BY sequence;"
psql "$DB_POSTGRES_DSN" -c "SELECT unnest(enum_range(NULL::journal_entry_type));"   # expect 14, no foreach:*
psql "$DB_POSTGRES_DSN" -c "SELECT state, claimed_by, claimed_at FROM workflows WHERE workflow_id = '<id>';"
```

Duplicate `step:completed` rows for one `exec_id` at widely separated sequences are the A-07
signature: one extra pair per restart.

Object-store layout (key builders live at the top of each postgres repo file):

| Key | Built by |
| --- | --- |
| `workflows/{id}/journal/{seq}/{input,result,data}.json` | `postgres/journal.go:28` |
| `workflows/{id}/output.json` | `postgres/workflow.go:28` |
| `workflows/{id}/execution-snapshot.json` | `workflow_handler.go:414` (bypasses the repository layer) |
| `workflows/{id}/trace/{execID}/{input,output}.json` | `postgres/trace.go:29` |
| `awakeables/{id}/result.json` | `postgres/awakeable.go:29` |
| `schemas/{id}/definition.json`, `schemas/{id}/v{n}/definition.json` | `postgres/graph.go:32`, `:16` |

With `OBJECT_STORE_DRIVER=filesystem` these are plain files under `OBJECT_STORE_FS_BASE_PATH`
(default `./data/fuse`) — the easiest layout to inspect while debugging replay.

HTTP read surface: `GET /v1/workflows/{id}/snapshot` serves the persisted snapshot if
`snapshot_ref` is set, otherwise builds one **live** from `journalRepo.LoadAll` +
`BuildExecutionSnapshot` (`internal/handlers/get_workflow_snapshot.go:59-98`) — the closest thing
to a mid-flight read model that exists today, though its `aggregatedOutputs` comes from a
repo-reconstructed `Workflow` and is therefore empty under Postgres (C-09).
`GET /v1/workflows/{id}/trace` reads `traceRepo.FindByWorkflowID` only and 404s until the run
terminates — snapshot and trace are persisted solely from `sendWorkflowCompleted`
(`workflow_handler.go:518-519`).

## Related ADRs

| ADR | Title | Why it matters here |
| --- | --- | --- |
| 0010 | Durable execution via an append-only journal | The governing decision. A change to replay semantics amends or supersedes it. |
| 0011 | Thread model for fork/join and ForEach iteration | Dynamic thread allocation, the ForEach limitation. |
| 0018 | High availability: claims, clustering, and schema replication | Who re-drives a crashed run, and the lease semantics. |
| 0019 | Externalize large payloads to a pluggable object store | The `*_ref` contract, and its accepted "no GC, orphan objects" cost. |
| 0020 | Observability: Prometheus metrics, OpenTelemetry tracing, execution traces | What the trace projection is for. |
| 0022 | Retry and error-handling model | `HandleNodeFailure`, `RetryTracker`, error edges. |
| 0023 | Timeout enforcement via actor timers | Why every deadline is in-memory `SendAfter` and dies with the process (C-06). |
| 0032 | Sub-workflow composition | `SubWorkflowRef` and parent notification. |
| 0003 | In-memory repositories by default, Postgres optional | Why two implementations exist and must be kept behaviourally identical. |

Coding rules that bind changes here: `.agents/rules/03-actor-patterns.mdc`,
`05-repositories.mdc`, `07-testing.mdc`, `12-quality-gates.mdc` (all `alwaysApply`). Author
guidance under `.agents/` only — `.claude/` and `.cursor/` are symlinks (ADR-0009).

Quality gate, in this order: `make lint && make build && make test`. For anything touching the
journal schema, also `DB_POSTGRES_DSN=... make test-functional` — the memory driver will not catch
a sequence collision or an ENUM violation.

$ARGUMENTS
