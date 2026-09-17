---
name: crash-resume-testing
description: Explains how to test FUSE durability for real — what the existing test tiers do and do not cover, how to stand up Postgres plus a filesystem object store, how to SIGKILL a real engine process at a reproducible point and assert on resume, and how to wire it into CI without slowing down `make test`. Use when building or extending the F-02 crash-resume harness, or when any change claims that a run survives a restart.
---

# crash-resume-testing

`Workflow.Resume()` has exactly **one** call site (`internal/actors/workflow_handler.go:156`) and
**zero** test callers — `grep '\.Resume(' --include='*_test.go'` returns nothing. Everything FUSE
claims about durability rests on a code path no test has ever executed. This pack is the knowledge
base for F-02 in the repo-root backlog (`BACKLOG_V2.md`). *If the maintainer has renamed it to
`BACKLOG.md`, the canonical file is the one titled "FUSE — product shape and backlog".*

F-02 blocks all of Tier A, and the backlog is explicit that **several scenarios will be red and
that is the point — do not fix them here.** A red scenario gets a linked task, not a patch.

## When to use

- Building the F-02 harness, or adding a scenario to it.
- Any PR that claims a run survives a restart (A-02, A-03, A-05, A-07, C-06, C-08, C-12).
- Deciding where a durability regression test should live across the test tiers below.

Read `.agents/skills/durable-execution-internals/SKILL.md` first — it explains what resume
actually does, which is what you are asserting against.

## Ground truth

| File | What it establishes |
| --- | --- |
| `internal/actors/workflow_handler.go:114-199` | `Init` — the trigger-vs-resume fork. The system under test. |
| `internal/workflow/workflow.go:187-306` | `Resume` / `replayJournalEntries` / `buildResumeAction` / `findPendingThreads`. |
| `tests/e2e/http.go`, `tests/e2e/disk.go` | Untagged helpers — polling, schema resolution. Reusable. |
| `tests/e2e/suite_e2e_test.go` | `RequireE2E`, `WorkflowsDirForTests`, `TestMain` and the `-workflows` flag. |
| `tests/e2e/workflow_fixture_e2e_test.go` | `UpsertSchema`, `TriggerExampleWorkflow`, `TriggerAndWaitTerminal`. |
| `docker-compose.yml` | The three profiles and the exact env each node gets. |
| `Makefile` | `test`, `test-functional`, `e2e-local`, `infra-up`, `ha-up`, `migrate`. |
| `.github/workflows/ci.yml`, `.../e2e.yml` | The `CI → E2E → CD` chain and where a new job attaches. |

## The existing test surface, and its exact holes

| Tier | Build tag | Command | Needs | Covers | Does **not** cover |
| --- | --- | --- | --- | --- | --- |
| Unit | *(none)* | `make test` (`./pkg/... ./internal/... ./tests/...`) | nothing | 100+ co-located `*_test.go`. Replay coverage is **two tests**: `TestReplayJournalEntries_SleepState` and `_CancelledState` (`internal/workflow/workflow_sleep_test.go:112,127`), which call `replayJournalEntries` directly and assert only the resulting `State`. `TestJournal_{Sleep,Awakeable,SubWorkflow}Entries` (`:18,41,64`) assert **write shape only**. | `Resume()` itself, `buildResumeAction`, `findPendingThreads`, any persistence round-trip. |
| Memory contract | *(none)* | `make test` | nothing | `tests/functional/*_test.go` untagged bodies — `contractTest<X>Repository` invoked as `TestMemory<X>Repository_Contract` (journal, workflow, graph, package, awakeable, environment, credential). | Postgres semantics. Memory `Append` never dedupes; Postgres has `UNIQUE (workflow_id, sequence)`. A replay bug that re-appends a sequence **passes silently here**. Also: **claims have no memory contract test** — `claim_repository_test.go` carries `//go:build functional`, so `make test` never exercises the claim contract (relevant to C-11). |
| Postgres contract | `functional` | `DB_POSTGRES_DSN=… make test-functional` | real PG | The same contract bodies against `postgres.New*Repository`, in schema `fuse_functional` (`tests/functional/postgres_test.go:22`). | Anything above the repository layer. No actor, no replay. **Skips silently** (`t.Skip`) when `DB_POSTGRES_DSN` is unset — green means nothing without the DSN. |
| E2E fast | `e2e` | `E2E_API_URL=… go test -tags=e2e ./tests/e2e` | running stack | `WorkflowResilienceSuite` — error edges, automatic retry, parallel retry, node timeout, all **in-process** failures with the engine alive throughout. `WorkflowPersistenceSuite` — trigger, finish, re-query, schema/package round-trip. | Process death. Nothing kills anything. The backlog names `workflow_resilience_suite_e2e_test.go` specifically: in-process failure is a **different thing** from crash-resume. |
| E2E slow | `e2e && e2e_slow` | `go test -tags="e2e e2e_slow" …` | running stack | `WorkflowOrchestrationSuite` — `sleep-test`, `subworkflow-test`, `awakeable-test`, `merge-strategy-test`, `timed-cond-test`. `WorkflowIntegrationSuite` — mermaid DAGs, `github-request-example`, `full-foundation-test`. | Restart. And note this tier **only runs on `main`** in CI (`e2e.yml`, `head_branch == 'main'`), so the durable primitives F-02 must kill mid-flight are today covered by a suite that never gates a PR. |

**No example workflow uses `system/foreach`** — `grep -rl foreach examples/workflows/` returns
nothing (the `e2e/` overlay included). A ForEach scenario needs a new schema, and under Postgres it
is expected to fail before it ever reaches replay because the four `foreach:*` strings are missing
from the `journal_entry_type` ENUM (C-08). The ENUM gap is confirmed; the runtime failure is read
off the code, not executed — see the caveat under "Worth adding beyond the seven".

## In-process failure vs real-process crash

These are different systems under test and conflating them is the mistake F-02 exists to correct.

| | In-process (what exists) | Real-process (what F-02 needs) |
| --- | --- | --- |
| What dies | one node function returns `FunctionError`, or a `SendAfter` timeout fires | the OS process |
| Actor tree | survives; the handler keeps its heap | destroyed — `forEachStates`, `iterThreadToForEach`, `RetryTracker`, every `SendAfter` timer, the OTel root span |
| Path exercised | `HandleNodeFailure` → retry / error edge | `WorkflowHandler.Init` resume branch → `LoadAll` → `LoadFrom` → `Resume` |
| Durable state | irrelevant | is the entire contract |
| Sufficient driver | memory | **Postgres + a shared object store only** |

**A crash test under `DB_DRIVER=memory` is meaningless.** `MemoryWorkflowRepository.Get` returns
the live `*Workflow` pointer with populated threads/auditLog/aggregatedOutput; the Postgres driver
returns a fresh empty one rebuilt from the active graph definition. Resume behaves fundamentally
differently. Likewise `OBJECT_STORE_DRIVER=memory` dies with the process, so every
`input_ref`/`result_ref` fetch fails and `LoadAll` aborts — that is F-03's failure mode, not a
resume test.

## Standing up the infrastructure

Three compose profiles, one file. Confirmed properties that matter for a kill test:

| Profile | Command | Nodes | `restart:` | Object store |
| --- | --- | --- | --- | --- |
| `infra` | `make infra-up` (`--profile infra up -d`) | none — engine runs on the host | n/a | rustfs S3 on `:9000` |
| `ha` | `make ha-up` (`--profile ha up --build -d`) | 3, built from source, `:9091-9093`, nginx LB `:9090` | **`unless-stopped`** — will fight a kill test | s3 |
| `e2e` | `docker compose --profile e2e up -d --wait` | 3, from `fuse-app:test`, `:9091-9093` | **`no`** — a killed container stays down until you start it | s3, bucket `fuse-e2e` |

`infra` gives PG 17 on `:5432` (`fuse`/`fuse`/`fuse`), rustfs on `:9000`
(`rustfsadmin`/`rustfsadmin`), etcd on `:2379`. The `e2e` profile additionally runs `fuse-pg-init`
(`CREATE SCHEMA IF NOT EXISTS fuse_e2e`) and a dedicated `fuse-migrate` init container that the
nodes gate on with `service_completed_successfully` — the right shape when concurrent DDL matters,
since the `ha` profile has no migrate init container and every node runs `RunMigrations` itself at
boot (`internal/app/di/database.go:58`).

**Recommended F-02 setup: `infra` profile + the engine as a host process.** You own the process
lifecycle, you can `kill -9` a PID with no container runtime in the way, and you can use the
filesystem object store so the payload graph is inspectable on disk.

```bash
make infra-up
export DB_DRIVER=postgres
export DB_POSTGRES_DSN='postgres://fuse:fuse@localhost:5432/fuse?sslmode=disable&search_path=fuse_crash'
export OBJECT_STORE_DRIVER=filesystem
export OBJECT_STORE_FS_BASE_PATH=/tmp/fuse-crash-store
psql 'postgres://fuse:fuse@localhost:5432/fuse?sslmode=disable' -c 'CREATE SCHEMA IF NOT EXISTS fuse_crash'
make migrate            # builds bin/fuse, then ./bin/fuse migrate  (rebuild is part of the target)
./bin/fuse server -l debug -p 9090 &
```

Migrations honour whatever `search_path` the DSN carries — that is how `fuse_functional` and
`fuse_e2e` coexist in the same database. Give the crash harness its own schema so it does not
collide with either. Leave `HA_ENABLED` at its default `false` for the base scenarios: with HA off
the **only** recovery is the one-shot `RecoverWorkflows` message sent from `Fuse.Start`
(`internal/app/fuse.go:242`, send at `:248-249`) at boot, which is exactly the path you want to
assert on. Turn HA on only for a claim-specific
scenario, and then account for `HA_LEASE_TIMEOUT` (default 30s, 15s in the e2e profile): after a
SIGKILL `WorkflowClaimActor.Terminate` never runs, so `ReleaseWorkflows` is skipped and the dead
node's `claimed_by`/`claimed_at` stay on the row until the lease expires.

## Killing the process deterministically

`SIGKILL` only — `kill -9 <pid>`, or `docker kill --signal=KILL <container>`. `SIGTERM` runs the
graceful shutdown path (`SHUTDOWN_TIMEOUT`, `ReleaseWorkflows`, `Terminate` hooks) and tests
nothing. On restart, re-exec the same `bin/fuse server` with the same environment; the harness
asserts across the boundary.

**Choose the kill point by observable state, never by `time.Sleep`.** Three reproducible levers,
best first:

1. **Park the run in `sleeping`.** `system/sleep` and `system/wait` both call
   `SetState(StateSleeping)` and then `persistWorkflowState()` **before** arming their `SendAfter`
   (`workflow_handler.go:749-763`, `:765-795`). Poll `GET /v1/workflows/{id}` until it reads
   `sleeping` and the journal is guaranteed flushed. `tests/e2e/http.go:189`
   `WaitForWorkflowStatus(client, baseURL, wfID, "sleeping", timeout)` already does the polling.
   This is the cleanest kill point in the engine.
2. **Poll the journal directly.** `SELECT entry_type FROM journal_entries WHERE workflow_id = $1`
   until the specific `step:completed` you want to crash after appears. Every state-visible
   transition is followed by `persistJournal`/`persistWorkflowState`, so a row in the table is a
   hard happens-before edge. Use this for "kill after a completed step" and for the multi-thread
   scenario (wait for exactly N `thread:finished` rows).
3. **Park on an external callback.** An async node returns `NewFunctionResultAsync()` and the run
   waits for `POST /v1/workflows/{workflowID}/execs/{execID}`. Beware: the async result is
   journaled as `step:completed` before the run parks (see Traps), so the DB status stays
   `running` — poll for the `step:completed` row of that exec instead.

Anti-pattern: killing after a fixed sleep. Runs finish in milliseconds under memory-free nodes, and
the whole test becomes a race against the engine.

Do **not** reuse `TriggerAndWaitTerminal` (`workflow_fixture_e2e_test.go:30`) in a crash harness.
It silently re-triggers up to 3 times when a workflow is stuck in `"untriggered"` — a quarantined
HA flake workaround. In a crash test `"untriggered"` may be the thing under test, and a fresh
`workflowId` per retry destroys the assertion.

## Assertion strategy

Assert on the durable record, not on "it finished". A run that completes can still have corrupted
its journal on the way.

| Property | How to assert | Guards |
| --- | --- | --- |
| **Journal length stable across three restarts** | `SELECT count(*) FROM journal_entries WHERE workflow_id = $1` before and after each restart, with the run parked and idle at each sample. Expect **equal**. | A-07 — replay's `SetResultFor` appends a fresh `step:completed` per previously-completed step, and `Init`'s `persistWorkflowState` flushes them. Growth is linear in restarts. |
| **No duplicate side effects** | `SELECT count(*) FROM sub_workflow_refs WHERE parent_workflow_id = $1` — expect exactly 1 per `system/subworkflow` node. `SELECT count(*) FROM awakeables WHERE workflow_id = $1` — expect 1 per gate. | A-02. Both are expected to be duplicated today — `handleSubWorkflowAction` mints a fresh `workflow.NewID()` and `handleSystemWait` a fresh `uuid.New()` on the re-dispatch path. **This is read off the code, not observed**; F-01.2 in the backlog exists to verify exactly the subworkflow half, and its "done when" allows for "it did not reproduce and why". |
| **Trace stability** | `GET /v1/workflows/{id}/trace` after terminal state; compare step count and per-step `duration`. | A-07's real symptom is not duplicated step rows (`BuildTrace` only creates one on `step:started`, which replay does not duplicate) but recomputed `CompletedAt`/`Duration` against the **replay** timestamp, and `Status` forced back to `completed`. |
| **Snapshot timeline stability** | `GET /v1/workflows/{id}/snapshot`, compare `timeline` length. | The timeline appends **every** entry (`execution_snapshot_builder.go:32`), so it grows by a full copy per restart. |
| **Sleep remainder honoured** | Record wall-clock before the kill; assert the run completes at roughly the original deadline, not `kill_time + full_duration`. | A-02 — `handleSystemSleep` re-parses `Args["duration"]` and re-arms the **full** duration. |
| **Pre-crash awakeable token still resolves** | Capture the awakeable id before the kill (today: from `journal_entries.data_ref` → `data.json`, or the `awakeables` table — there is no route, C-07); after restart `POST /v1/awakeables/{id}/resolve` and expect success. | This one is expected to **pass**: `resolve_awakeable.go:84-85` uses the awakeable's own stored `ExecID`/`ThreadID`, and `replayPendingThread` reuses the original execID. So the predicted failure is the **extra** row, not the old token — but that prediction is code reading, and this scenario is how you find out. |
| **Exactly one advance per gate** | After resolving, assert the thread advanced once — `count(*)` of `step:started` for the downstream node is 1. | A-02's double-advance. |

**Exactly-once vs at-least-once — do not assert what has not been decided.** Today's behaviour is
at-least-once with **no dedup beneath it**: `replayPendingThread` re-issues a pending step with the
original execID and no journal marker, and
`internal/handlers/async_function_result.go` performs **no** pending-state or duplicate validation.
**A-03 is the open decision** — (a) at-least-once with a stable idempotency key the SDK dedupes on,
or (b) replay-aware re-arming that does not re-dispatch — and it must be settled in an ADR and
written into the B-01 protocol spec. Until then, write the assertion as *"the node was invoked N
times and the run's observable outcome is X"* and record N, rather than asserting "exactly once".
Making the harness encode a guarantee nobody has chosen is how the decision gets made by accident.

## The seven F-02 scenarios

Each is its own test. Expected-red is normal; link the owning task rather than fixing it. **The
"expected today" column is a prediction derived from reading the code — no scenario here has been
executed.** A scenario that comes out green is a real result: close the premise and say so, per the
backlog's "verify before fixing".

| # | Scenario | Kill point | Assert | Expected today |
| --- | --- | --- | --- | --- |
| 1 | Kill after a completed step | poll `journal_entries` for the target `step:completed` | that node is not re-executed; journal has no second `step:started` for it | should pass, but journal grows by a duplicate `step:completed` → **A-07** |
| 2 | Kill with a pending async node | poll for the async exec's `step:completed` (it parks as completed) | the run resumes and the pre-crash callback URL still advances it exactly once | fragile — the parked step is **not** seen as pending, so resume does not re-arm it; the run waits on the external callback with no timer → informs **A-03** |
| 3 | Kill with a pending `system/subworkflow` | poll for `subworkflow:started` | **exactly one child after restart** — one row in `sub_workflow_refs`, one child run | predicted **red — A-02** (and this is F-01.2). A second child is minted at `workflow_handler.go:902`; the follow-on — both children completing and both notifying the same `ParentExecID` — is inference, and this scenario is the thing that settles it |
| 4 | Kill with a pending `system/sleep` | status `sleeping` | wake-up fires at the original deadline (remainder, not full duration) | **red — A-02** |
| 5 | Kill with a pending awakeable | status `sleeping` | pre-crash token still resolves; exactly one row in `awakeables`; one advance | token expected to resolve; predicted **red on the duplicate row and the reset timeout — A-02** (`handleSystemWait` re-mints at `workflow_handler.go:728`, re-arms the full timeout at `:789-794`) |
| 6 | Multi-thread run killed between thread completions | poll for exactly N `thread:finished` rows with no step in flight | every branch resumes | **premise is inference — C-12.** Loss is only on the zero-pending branch (`workflow.go:229-238`); the pending branch fans out correctly. `Next()` has side effects, so discarded actions have already journaled their `step:started` and the *next* restart picks them up — expect it to look flaky across one restart and self-heal across two. Verify before filing |
| 7 | Three restarts on one run | park, kill, restart, ×3 | journal length and `BuildTrace` output identical after each cycle | **red — A-07**, compounding |

**Worth adding beyond the seven:** a run containing `system/foreach` under Postgres. The expectation
is that it fails before it ever reaches replay — the `foreach:*` types are absent from the
`journal_entry_type` ENUM (confirmed: 14 values across all 11 migrations, none of them `foreach:*`),
so the first `persistJournal` after the foreach node starts should fail the transaction and, because
`persistJournal` skips `MarkPersisted` on error (`workflow_handler.go:467-471`), the run's journal
freezes permanently. **That runtime consequence is derived from reading the code and has not been
executed against a live Postgres** — confirm it before spending time on ForEach replay logic (C-08).

## Wiring it into CI without slowing `make test`

**None of this exists yet — it is what F-02 has to add.** The only confirmed facts in this section
are the shape of the current pipeline; everything phrased as an instruction is a recommendation to
the implementer, not a description of the repo.

The chain is `CI → E2E → CD`, each triggered by the previous workflow's `workflow_run` (`ci.yml` on
push/PR to `main`; `e2e.yml` on `workflows: ["CI"]`; `cd.yml` on `workflows: ["E2E"]`,
`branches: [main]`). Attach the harness to `.github/workflows/e2e.yml`, not to `ci.yml`.

- **Give it its own build tag**, e.g. `//go:build e2e && e2e_crash`, mirroring the existing
  `e2e && e2e_slow` pattern. Untagged files in `tests/e2e/` (`http.go`, `disk.go`, `doc.go`) stay
  untagged so the helpers keep compiling under plain `go test ./tests/e2e`; only the test bodies
  carry the tag. This keeps `make test` (which compiles `./tests/...` with **no** tags) completely
  unaffected — it will pick up nothing new.
- **New job in `e2e.yml`**, modelled on `e2e-fast`: checkout `workflow_run.head_sha`, Go 1.26,
  build `fuse-app:test`, bring up infra, run with an explicit `-timeout`, dump
  `docker compose logs --tail=200` on failure, `down -v` in `always()`.
- **Gate it like `e2e-fast`, not like `e2e-slow`.** `e2e-slow` only runs when
  `head_branch == 'main'`, i.e. after merge — a durability regression would land on main before
  anyone saw it. F-02's whole value is catching it on the PR.
- **Budget the runtime.** `e2e-fast` uses `-timeout 10m` and `e2e-slow` `-timeout 15m` (`make
  e2e-local` uses 15m). Seven kill cycles plus process restarts is minutes, not seconds; keep sleep
  durations in the fixtures short and prefer journal polling over wall-clock waits. Note the
  existing `sleep-test` sleeps **5s** in `examples/workflows/sleep-test.json` but **500ms** in the
  e2e overlay `examples/workflows/e2e/sleep-test.json` — copy the overlay's magnitude, and be aware
  a 500ms sleep is a poor kill window.
- **Do not add it to `scripts/pre-commit-gates.sh`.** That script runs `make lint && make build &&
  make test` and is the local gate; a Docker-dependent suite does not belong in it.
- `.github/workflows/e2e.yml` and `cd.yml` trigger on `workflow_run`, which means they execute from
  the **default-branch** definition of the workflow file. Changes to `e2e.yml` on a PR branch are
  not exercised by that PR — expect one merge before the new job actually runs.

## Traps

- **An async-parked step is journaled as `step:completed`.** `NewFunctionResultAsync()` carries
  `Output.Status = FunctionSuccess` (`pkg/workflow/fn_result.go:43-49`) and
  `handleMsgFunctionResult` calls `SetResultFor` **before** checking `Result.Async`
  (`workflow_handler.go:275-281`). So `findPendingThreads` never sees an in-flight async node, and
  the DB status stays `running`, not `sleeping`. Scenario 2 must poll the journal, not the status.
- **`t.Skip` makes an unconfigured suite green.** `tests/functional/postgres_test.go:testDSN` skips
  without `DB_POSTGRES_DSN`; `RequireE2E` skips under `testing.Short()`. Assert in CI that the
  expected number of tests actually **ran**, or a misconfigured job passes forever.
- **`E2E_API_URL` must be set.** `DefaultAPIURL` is `http://localhost:9090` (`http.go:15`) but the
  e2e stack publishes `9091-9093`; the LB on `:9090` only exists in the `ha` profile. Every
  make target and CI job sets `E2E_API_URL=http://localhost:9091` explicitly.
- **`examples/workflows/e2e/` silently shadows production schemas** of the same name during e2e
  (`sleep-test`, `sum-rand-branch`, `timed-cond-test`, `timeout-test`, via
  `ReadSchemaFileWithOverlay`). A local reproduction with the production JSON can behave
  differently from CI.
- **A fresh clone cannot build.** `docs/docs.go` and `docs/swagger.{json,yaml}` are gitignored but
  `internal/actors/mux_server.go` blank-imports the generated `docs` package. Run `make swagger`
  before `make build`/`make test`.
- **Results are cached aggressively.** `go clean -testcache` before trusting a green replay run;
  the e2e targets already pass `-count=1`.
- **Post-SIGKILL recovery is not instant under HA.** `ClaimWorkflows` will not take a row until
  `claimed_at` is older than `HA_LEASE_TIMEOUT`, and the sweep claims a hardcoded batch of 10 per
  `HA_CLAIM_SWEEP_INTERVAL` tick (`workflow_claim_actor.go:110`). Budget the timeout accordingly,
  or run the base scenarios with `HA_ENABLED=false`.
- **`make test` runs without `-race`.** No gate enables it. Run `go test -race ./internal/... ./pkg/...`
  by hand when a fix touches actors or the memory repositories.

## Conventions to follow

- Suite shape: a `testify` `suite.Suite` with `client *http.Client; baseURL string; workflowsDir string`,
  a `func TestXxxSuite(t *testing.T) { t.Parallel(); suite.Run(t, new(XxxSuite)) }` entry point, and
  a `SetupSuite` calling `RequireE2E(s.T())` + `WorkflowsDirForTests(s.T())` then looping
  `UpsertSchema` over the schema ids. Crash suites should **not** be `t.Parallel()` against each
  other if they share one engine process.
- Test method naming `TestThing_ObservableOutcome`, every assertion carrying a message that states
  the expectation. Literal `// Arrange` / `// Act` / `// Assert` blocks, `require` for anything that
  must stop the test and `assert` for the rest (`.agents/rules/07-testing.mdc`).
- Lint and the helper files: `.golangci.yml` declares `issues.exclude-rules` dropping `errcheck` and
  `gosec` for `path: _test\.go`, but the untagged helpers in `tests/e2e/` (`http.go`, `disk.go`) are
  **not** `_test.go` files, so both linters apply to them in full. Put helper code there in the same
  shape they already use — `defer func() { _ = resp.Body.Close() }()` (`http.go:88,103,122`) and
  `//nolint:gosec` on a deliberate path join (`disk.go:105`). (Whether the `_test.go` exclusion is
  still honoured under golangci-lint v2's schema is unverified here — write test bodies defensively
  and let `make lint` arbitrate.)
- The reproducing test lands in the same PR as the fix, **failing in the first commit**
  (`BACKLOG_V2.md`, "Rules for the implementing agent"). One task, one PR. Bugs found along the way
  get appended to the backlog with evidence, not fixed inline.
- Author guidance under `.agents/` only — `.claude/` and `.cursor/` are symlinks (ADR-0009).

## Commands

```bash
make swagger                                              # first, on a fresh clone
make lint && make build && make test                      # the mandated gate, in this order
go test -v -run 'TestReplayJournalEntries' ./internal/workflow/   # the only existing replay tests
go clean -testcache

make infra-up                                             # PG 5432 + rustfs 9000 + etcd 2379
make migrate                                              # build + ./bin/fuse migrate (needs DB_POSTGRES_DSN)
DB_POSTGRES_DSN='postgres://fuse:fuse@localhost:5432/fuse?sslmode=disable' make test-functional

make e2e-local                                            # build image, --profile e2e up --wait, tags=e2e vs :9091, down -v
E2E_API_URL=http://localhost:9091 go test -tags=e2e ./tests/e2e -v -count=1 -timeout 10m
E2E_API_URL=http://localhost:9091 go test -tags="e2e e2e_slow" ./tests/e2e -v -count=1 -timeout 15m
docker compose --profile e2e logs --tail=200 && docker compose --profile e2e down -v

make ha-up / make ha-down                                 # 3 nodes from source; note restart: unless-stopped
```

## Related ADRs

`docs/adr/0010-durable-execution-journal-and-replay.md` (the contract under test — note it
describes intent, not current behaviour), `0018-high-availability-and-clustering.md` (claims and
leases, and why a SIGKILL'd node's claim lingers), `0019-object-store-payload-externalization.md`
(why the object store must outlive the process), `0023-timeout-enforcement-model.md` (why every
deadline is an in-memory `SendAfter` and therefore dies with the process),
`0021-deployment-and-delivery-architecture.md` (the CI → E2E → CD chain and the compose profiles),
`0003-in-memory-repositories-by-default.md` (why a memory-driver green proves nothing here).

$ARGUMENTS
