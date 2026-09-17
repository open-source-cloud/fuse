---
name: remote-node-protocol
description: Inventories every primitive the FUSE extension protocol will be built on — the transport enum nobody reads, the async result constructor, the unvalidated callback route, ExecutionInfo's real field list, the metadata-only registration dead end, retry/timeout/coercion machinery a remote call inherits — and separates that shipped reality from the eight decisions B-01 must settle. Use when working B-01, B-02, B-05, B-08 or C-10, when designing anything that crosses the engine↔worker boundary, or before claiming that FUSE "supports HTTP transport".
---

# remote-node-protocol

The extension protocol is FUSE's product pillar and it is **half-built and undocumented**. This pack
is the inventory an SDK-facing spec must be written on top of.

Backlog ids are from the repo-root backlog (`BACKLOG_V2.md`). *If it has been renamed to
`BACKLOG.md`, the canonical file is the one titled "FUSE — product shape and backlog."*

> **The two halves are two different kinds of claim.** **§ TODAY** is shipped behaviour, every line
> traceable to a `file:line`. **§ TO BE DECIDED** is net-new design owned by B-01 — nothing there
> exists. Quoting a § TO BE DECIDED clause as engine behaviour is the worst failure mode in this area.

## When to use

- **B-01** writing the spec (route to the `fuse-protocol-spec-author` agent), **B-02** implementing
  the remote transport, **B-05** the conformance suite, **B-08** worker lifecycle, **C-10** the
  per-node idempotency key, and **A-03** (replay of pending remote steps), which the spec must state.
- Any time someone reads `pkg/transport.HTTP` and concludes HTTP transport exists.

**Out of scope here:** the in-process function contract (`function-package-authoring`), replay and
journal mechanics (`durable-execution-internals`), spans (`observability-tracing`). **B-06/B-07 —
the TypeScript and PHP SDKs — live outside this repo entirely.**

## Ground truth

| File | Authoritative for |
| --- | --- |
| `pkg/transport/type.go` | The entire transport enum — 11 lines, no logic |
| `internal/packages/transport/function.go` | `FunctionTransport`, the seam a remote transport implements |
| `internal/packages/transport/internal.go` | The only implementation; where `Finish` is bound |
| `internal/packages/loaded_package.go` / `registry.go` | Registration mapping, the executable/metadata-only branch, the non-downgrade rule |
| `pkg/workflow/execution_info.go` | Every field a function receives — five |
| `pkg/workflow/fn_result.go` / `fn_output.go` | The result constructors and the one status enum |
| `internal/handlers/async_function_result.go` | The external callback route |
| `internal/messaging/execute_function.go` | The closest existing analogue to an invocation envelope |
| `internal/typeschema/parse.go` | The complete set of coercible declared types |
| `internal/idempotency/store.go` | The idempotency contract (trigger-level only, ADR-0017) |
| `docs/adr/0024`, `0022`, `0023`, `0017`, `0027` | Registry contract, retry/error, timeouts, idempotency scope, the `ExecutionInfo` prohibition |

---

## § TODAY — every primitive that exists

### The transport layer

| Primitive | Location | Reality |
| --- | --- | --- |
| `transport.Type` enum | `pkg/transport/type.go:5` | Wire-level, public, JSON-visible |
| `HTTP Type = "http"` | `pkg/transport/type.go:9` | **Never read by production code.** Its only appearances in the tree are 8 inert fixture values in `tests/functional/package_repository_test.go` |
| `gRPC Type = "grpc"` | `pkg/transport/type.go:10` | **Lowercase `g` → unexported.** Unreachable outside `pkg/transport`. Dead code, not a stub |
| `Internal` | `internal/packages/transport/type.go:6` | A *different* package. The only transport that executes |
| `FunctionTransport` | `internal/packages/transport/function.go:10-18` | `Execute(actor.Handle, *ExecutionInfo)` + `ExecuteSync(*ExecutionInfo)` |
| `InternalFunctionTransport` | `internal/packages/transport/internal.go:35` | The only implementation |

No production code branches on `transport.Type` at all — the only two comparisons in the tree are
`loaded_package.go:112` and `agent_tools.go:101`, both against `Internal`.

### Registration — and the trap that makes `Transport` metadata meaningless

`MapToRegistryPackage` (`internal/packages/loaded_package.go:63`) builds the registry-side
`FunctionMetadata` with **`Transport: transport.Internal` hardcoded at L70**, whatever the incoming
wire metadata declared. It then branches at **L112**:

```go
if function.Metadata.Transport == transport.Internal && function.Function != nil {
    functions[functionID] = NewLoadedInternalFunction(functionID, metadata, function.Function)
} else {
    functions[functionID] = NewLoadedFunction(functionID, metadata) // Transport == nil
}
```

⚠️ **Consequence, confirmed by reading:** `packages.FunctionMetadata.Transport` is **always**
`"internal"` in the registry. The only working discriminator between an executable and a
metadata-only entry is `LoadedFunction.Transport == nil` (`loaded_function.go:26-30`). Any B-02/B-03
predicate written against `Metadata.Transport` will silently match everything. This also makes the
second clause of `isExposableTool` (`agent_tools.go:101`) vacuously true today.

Dispatch of a metadata-only function fails at `loaded_package.go:41` / `:56` with the verbatim text:

```
function %s has no transport (package likely loaded from persistence without its code-backed function)
```

`Registry.Register` (`registry.go:47-65`) will **discard** any incoming function whose `Transport` is
nil in favour of a previously registered executable entry, so a remote registration that does not
produce a non-nil `Transport` is silently thrown away. The rule is load-bearing and regression tested
(`internal/packages/loaded_package_test.go:83`).

### What a function receives, and what it returns

`pkg/workflow/execution_info.go:14-23` — **five fields**: `WorkflowID`, `ExecID`, `Environment`,
`Input`, `Finish`. No `context.Context`, no nodeId, no threadId, no attempt, no deadline, no
idempotency key. `Finish` is nil until a transport binds it (`internal/packages/transport/internal.go:47`).

`ExecID` (`pkg/workflow/exec_id.go:10`) is a UUIDv8 with the thread id packed into bytes 6–7;
`ExecID.Thread()` (L23) recovers it via `hex.DecodeString` whose error is **discarded**, then indexes
`raw[6]`/`raw[7]`.

| Result constructor | `Async` | Meaning |
| --- | --- | --- |
| `NewFunctionResult(status, data)` (`fn_result.go:10`) | false | general sync result |
| `NewFunctionResultSuccess()` / `…SuccessWith(data)` | false | success |
| `NewFunctionResultError(err)` (`:36`) | false | returns `(result, nil)` — a **nil Go error** |
| `NewFunctionResultAsync()` (`:44`) | true | "I will call `Finish` later"; output is `success` with nil data |

The status enum is three values (`fn_output.go:6-10`), of which the engine branches on two:
`FunctionSuccess` and `FunctionError`.

### The two async completion paths

**In-process (works).** Return `NewFunctionResultAsync()` and later call `execInfo.Finish(output)`.
`Finish` is bound by `InternalFunctionTransport.Execute` (`internal.go:47`) and delivers via
`sendAsyncFunctionResult` (`internal.go:18`) — `gen.Node.Send` to
`gen.Atom(actornames.WorkflowHandlerName(wfID))`, **by registered name, not PID** (`internal.go:12-17`).

**External callback (exists, unvalidated).**

```
POST /v1/workflows/{workflowID}/execs/{execID}
{"result": {"status": "success", "data": {}}}
```

Route: `internal/actors/mux_worker.go:79` — POST only, 10s timeout, `PoolSize: 3` (the route entry
spans `:76-86`).
Handler: `internal/handlers/async_function_result.go:53-85`.
Body DTO: `dtos.AsyncFunctionRequest{Result workflow.FunctionOutput}` (`internal/dtos/workflow.go:29`)
— one field, **no validate tag**, no attempt, no signature, no idempotency field.

⚠️ **Defect, do not present as the contract.** The handler reads two path params, binds JSON, and
`Send`s. It does **not** check that the workflow exists, that the exec is pending, that the exec
belongs to that workflow, that the execID is well-formed, or that a result was not already
submitted; it returns `200 {"code":"OK"}` whenever the actor `Send` succeeds. The e2e test documents
this (`tests/e2e/apis_e2e_test.go:201-217`: "workflow handler may not exist → 500 is acceptable for
smoke"). **B-02 owns fixing it.** The validation template to copy is
`internal/handlers/resolve_awakeable.go:66-73`, which 404s on missing and 400s on non-pending.

⚠️ **A duplicate callback advances the graph twice.** `Workflow.SetResultFor`
(`internal/workflow/workflow.go:422`) overwrites `entry.Result`, re-sets `aggregatedOutput`, and
appends a **second** `step:completed`; `handleMsgAsyncFunctionResult` then calls `Workflow.Next`
again. Its only guard is `a.isTerminalState()` (`workflow_handler.go:319`).

⚠️ **An async dispatch is journalled `step:completed` immediately.**
`handleMsgFunctionResult` calls `SetResultFor` at **`workflow_handler.go:275`, before** the
`if fnResultMsg.Result.Async` check at **`:277`**. So the moment a node returns
`NewFunctionResultAsync()`, the journal already carries a `step:completed` for it with the
placeholder empty output, and `aggregatedOutput[nodeID]` is set to nil. This inverts the naive
assumption behind A-03: `findPendingThreads` (`workflow.go:267`, `step:started` without completion)
will **not** see an in-flight remote call as pending. Settle it against the F-02 harness before
choosing replay semantics.

### What a remote node inherits for free

| Mechanism | Location | Notes |
| --- | --- | --- |
| Per-node retry | `internal/workflow/retry.go` (ADR-0022) | `RetryPolicy{MaxAttempts (0–100), Backoff{fixed\|exponential\|linear, InitialInterval, MaxInterval, Multiplier}}`; default 3 / exponential / 1s→30s / ×2 |
| Retry dispatch | `workflow.go:841 HandleNodeFailure` | Reuses the **same `ExecID`**; attempt count lives only in the in-memory `RetryTracker` (`retry_tracker.go`) — **A-08** owns journalling it |
| Error routing | `EdgeSchema.OnError` (`edge_schema.go`), `findErrorEdges` (`workflow.go:899`) | Fires only **after** retries are exhausted, and only `errorEdges[0]` is followed |
| Per-node timeout | `internal/workflow/timeout.go` `TimeoutConfig.Execution` (ADR-0023) | `startExecutionTimeout` (`workflow_handler.go:622`) → `ExecutionTimer` (`execution_timer.go`), in-memory `SendAfter`; zero = no timeout. Lost on restart — **C-06** |
| Timeout failure shape | `workflow_handler.go:551,564` | Synthesises `FunctionError` with `Data{"error": "execution timeout exceeded"}` and reuses the ADR-0022 failure path |
| Input coercion | `internal/typeschema/parse.go:20 ParseValue` | Exactly `string`, `int`, `float64`, `bool`, `[]byte`, `map[string]any`, plus `[]T` of those; anything else → `unsupported type: %s` |
| Schema binding | `internal/services/graph_service.go:295 populateNodeMetadata` | Splits `node.Function` at the **last** `/`, `Registry.Get(pkgID)`, `GetFunctionMetadata(fullID)` |

⚠️ **Coercion failures are swallowed.** Every failure branch in `inputMapping` (`workflow.go:652`)
and `applyFlowMapping` (`:727`) is a log line (`log.Error`, in one case `log.Warn`) followed by
`continue`, `break` or `return` — none returns an error to the caller. A type mismatch produces a
**missing parameter**, not a failed node. A declared capability schema therefore buys type
*coercion*, not type *enforcement*.

⚠️ Two shipped declarations already use a type `ParseValue` rejects: `headers`
(`internal/packages/functions/http/request.go:66-67`) and `data`
(`internal/packages/functions/system/wait.go:23`) both declare `Type: "map"`, not `"map[string]any"`.

### Idempotency, as it actually is

`internal/idempotency/store.go:7-19` — `Check` / `Set` / `Delete` / `CheckAndSet`. Keys are **opaque
strings**; the store knows nothing about execs, nodes or attempts. ADR-0017 states the scope limit
explicitly: dedup is at the **trigger** boundary.

| Producer | Key shape | Atomicity |
| --- | --- | --- |
| HTTP trigger | client-supplied `idempotencyKey` | ⚠️ non-atomic `Check` (`trigger_workflow.go:79`) then `Set` (`:104`) — two concurrent identical requests can both pass |
| Cron | `cron:<schemaID>:<RFC3339 minute>` (`cron_scheduler.go:86`) | atomic `CheckAndSet` |
| Event | `evt:<schemaID>:<type>:<source>:<sha256(json(data))[:16]>` (`event_trigger.go:120-124`) | atomic `CheckAndSet` |

Nothing derives a key from `(workflowID, execID, attempt)`. That is **C-10**.

### Registration over the API is lossy in both directions

`internal/dtos/package.go` has **no `Edges` field on `InputMetadataDTO` or `OutputMetadataDTO`**
(grep confirms zero hits for `Edges` in that file). So `FromPackagedFunctionDTO`
(`internal/dtos/package.go:150`) yields a function with **zero declared input or output edges** — an
API-registered capability cannot declare conditional or error routing at all — and a `GET → PUT`
round-trip loses data. `FunctionMetadataDTO.Transport` even carries the swagger example `"sync"`
(`internal/dtos/package.go:36`), which is not a valid `transport.Type`, and
`FromPackagedFunctionDTO` casts that string straight through (`:178`) with no validation.
**No backlog task names the dropped-edges defect.** B-01 decision 7 (capability declaration) is its
natural home, but that is this pack's inference and not something the backlog says — file it with
evidence rather than fixing it inside another task. This paragraph is the single source for the
registration-loss inventory; `function-package-authoring` points here rather than restating it.

⚠️ **Per-function concurrency and rate limiting are unreachable from any registration path.**
`packages.FunctionMetadata` declares `Concurrency` and `RateLimit`
(`internal/packages/function_metadata.go:14-15`) and `WorkflowFunc.HandleMessage` reads both
(`workflow_func.go:102-109`), but the wire type `workflow.FunctionMetadata`
(`pkg/workflow/metadata.go:6-10`) has only `{Transport, Input, Output}`, the DTO has neither field,
and `MapToRegistryPackage` never sets them — so both are always nil for every registered package.
This is not "DTO loss"; it is a field pair with a consumer and no producer.

⚠️ `PUT /v1/packages/{packageID:.+}` **ignores its path parameter**: `register_package.go:61` reads
it and uses it only in log/error messages; the saved id comes from the JSON body. There is **no
DELETE route for packages** — that route declares `GET, PUT` only (`mux_worker.go:120-121`; other
resources do declare `DELETE`) — though `postgres/package.go:162` implements `Delete`.

### Auth: greenfield

`docs/API.md` says verbatim "**Authentication** — Not required today". There is no middleware in
`internal/actors/mux_server.go`; `Handler.SendJSON` sets `Access-Control-Allow-Origin: *`
(`internal/handlers/handler.go:59`). The only credential machinery that exists is `pkg/secrets`:
`SecretValue` redacts in every String/JSON sink and only `Reveal()` yields plaintext;
`CredentialSecretName(id, field)` resolves to `cred/<id>/<field>` (`pkg/secrets/credential.go:15`);
`workflow.Credential` carries field **names** only.

---

## § TO BE DECIDED — the eight B-01 decisions and what constrains each

**None of this is built.** Each row names the engine constraint that makes the choice cheap or
load-bearing.

| # | Decision | Constraint from existing code | Cost to reverse |
| --- | --- | --- | --- |
| 1 | **Push or pull** | Nothing exists for pull: the only worker-facing inbound surface is the async-result callback route, and no route declared in `mux_worker.go` is a lease/poll/stream endpoint. Push matches "register your existing API as a node" | **Highest in the document.** Backlog recommends push first, pull later for locked-down networks |
| 2 | **Invocation envelope** | Closest analogue `messaging.ExecuteFunctionMessage` (`execute_function.go:12-20`) has 7 fields and lacks attempt, deadline, callback URL and idempotency key; `traceparent` does exist but one level up, on `messaging.Message.TraceCarrier` (`message.go:44`), not on the envelope. Only 4 of the 7 survive into `ExecutionInfo`. `nodeId` exists only as `entry.FunctionNodeID` on audit-log/journal entries; `threadId` is on the message and derivable via `ExecID.Thread()` | Medium — additive fields are cheap, renames are not |
| 3 | **Ack semantics** | One bit: `FunctionResult.Async`. Async parks the run at `workflow_handler.go:277` — but see the `SetResultFor`-before-the-check trap above | Medium |
| 4 | **Completion taxonomy** | **No representation today.** One status plus a free-form `Data["error"]`; timeout, rate-limit rejection and "no transport" all synthesise the identical shape. Needs a **new wire field**, not a convention | High — retry safety depends on it |
| 5 | **Delivery guarantee (A-03)** | Retries reuse the same `ExecID`; manual `RetryNode` (`workflow.go:451`) mints a new one; attempt count is in-memory only. Requires A-08 before a key can include attempt | High — SDK dedup logic is downstream |
| 6 | **Auth, both directions** | 100% greenfield. Secret plumbing exists (`pkg/secrets`), transport-level identity does not | High — crosses a trust boundary |
| 7 | **Capability declaration** | Metadata-only packages **already** bind into graphs (`populateNodeMetadata`), so schema load is not the gap. Type set is capped by `ParseValue`'s six. Edge metadata is dropped by the REST DTO | Medium — schema shape is public |
| 8 | **Versioning** | No representation: `workflow.Package{ID, Functions, Tags}`. The only versioning is graph-schema versioning (`internal/workflow/versioned_schema.go`), pinned per run by A-05 | High — every SDK negotiates on it |

---

## B-02 implementation sketch (after B-01 is published)

1. **New transport** implementing `internal/packages/transport/function.go:FunctionTransport`, in a
   new file beside `internal.go`. Do **not** add a dispatch switch elsewhere —
   `LoadedPackage.ExecuteFunction` already delegates blindly to `function.Transport`.
2. **`NewLoadedRemoteFunction`**, a peer of `NewLoadedInternalFunction` (`loaded_function.go:17`),
   producing a **non-nil** `Transport` — otherwise `Registry.Register`'s non-downgrade rule discards it.
3. **`MapToRegistryPackage`**: stop hardcoding `Transport: transport.Internal` at **L70**, and give
   the `else` at **L118** a remote arm. Both edits are required; either alone leaves the registry
   unable to tell a remote capability from an internal one.
4. **Release the pool worker.** `WorkflowFunc.HandleMessage` calls `pkg.ExecuteFunction`
   synchronously (`workflow_func.go:136`) and the pool is `PoolSize: 3` **per workflow instance**
   (`workflow_func_pool.go:40`). A blocking remote call holds a third of that run's dispatch
   capacity for its whole duration. The transport must return `NewFunctionResultAsync()` and resume
   on callback — this is exactly **A-10**.
5. **Validate the callback.** Add existence / pending / duplicate / format checks in
   `internal/handlers/async_function_result.go`, modelled on `resolve_awakeable.go:66-73`, and
   remember `SetResultFor` is not idempotent.
6. **Inject `traceparent`.** Blocked on **A-04**: `workflow_func.go:96` is literally `_ = nodeCtx`
   and `ExecutionInfo` has no `context.Context`. ADR-0027 forbids re-adding a per-execution runtime
   *handle* as an `ExecutionInfo` field; plain data fields are a different question the A-04 ADR must
   settle.
7. **Persistence already indexes it.** `postgres/package.go:151` writes
   `package_functions(package_id, function_id, transport)`.
8. **Fix or scope the metadata gap.** Edges are dropped by the REST DTO in both directions; per-function
   `Concurrency`/`RateLimit` have a consumer (`workflow_func.go:102-109`) and no producer at all. A
   remote capability that cannot declare an error edge cannot participate in the ADR-0022 failure
   model, and one that cannot declare a concurrency cap will hammer the customer's endpoint.

## B-05 — the conformance-suite scenarios and what each pins

| Scenario | Pins | Existing anchor |
| --- | --- | --- |
| Sync result | `FunctionResult{Async:false}` reaches the handler | `workflow_func.go:172` |
| Async callback | `POST /v1/workflows/{id}/execs/{execID}` resumes the run | `workflow_handler.go:313` |
| Business error | routes to the **first** `OnError` edge, after retries | `workflow.go:899` |
| Infrastructure error | **indistinguishable today** — needs the B-01 taxonomy field | — |
| Retry with a stable key | backoff schedule + key stability across attempts | `retry.go DelayFor`; note the reused `ExecID` |
| Trace propagation | `traceparent` in and out | blocked on **A-04** |
| Deadline handling | what the worker does when its deadline passes | `timeout.go`; ADR-0023's caveat: the remote side keeps running |
| Schema validation | declared types coerce; mismatches must not silently vanish | `typeschema/parse.go`; `applyFlowMapping` swallow |
| Restart behaviour | per the **A-03** decision | `findPendingThreads` + the async `step:completed` trap |

Harness precedent: `tests/functional/package_repository_test.go:13 contractTestPackageRepository` is
the contract-test shape (one body, both drivers); `tests/e2e/` is the black-box HTTP shape
(`//go:build e2e`, polls `E2E_API_URL`, never boots a server).

## B-08 — worker lifecycle: the open questions

Nothing exists. Registration is a plain upsert (`package_service.go:80` — `Validate` → repo `Save` →
`Registry.Register`): no health check, no deregister, no drain, no TTL, no heartbeat, and no package
DELETE route. `FindAll`/`FindByID` with `Load: true` backfill the registry **only** `if !packageRegistry.Has(id)`,
so an already-registered capability is never refreshed by a GET — updates must go through `Save`.
"Worker unreachable" has no error class and would surface as a generic `FunctionError`, retried by
the node's `RetryPolicy`. Multiple workers behind one capability has no addressing model
(`workflow.Package` carries `{ID, Functions, Tags}`). Versioned rollout interacts with
`versioned_schema.go` and **A-05**.

## C-10 — the derived idempotency key

`ExecutionInfo` already carries `WorkflowID` and `ExecID`, so two of three components are present.
The third is the problem: automatic retries **reuse** the same `ExecID`, `RetryTracker` is in-memory
and never journalled as a number, and manual `RetryNode` mints a fresh `ExecID`. The attempt must be
threaded through `workflowactions.RetryFunctionAction.Attempt` (which already exists, set in
`HandleNodeFailure`) into `messaging.ExecuteFunctionMessage` and then into `ExecutionInfo` — that is
**A-08**'s dependency. Follow the shape of the existing deterministic keys (`cron:…`, `evt:…`).

## Commands (confirmed to exist)

```bash
make swagger                                   # REQUIRED on a fresh clone: docs/docs.go is gitignored but blank-imported (mux_server.go:19)
make lint && make build && make test           # the mandated gate, in this order
go test -v ./internal/packages/...             # registry, mapping, non-downgrade rule
go test -v -run TestRegister_DataOnlyCopy_DoesNotDowngradeExecutableFunc ./internal/packages/
go test -v ./internal/typeschema/ ./internal/idempotency/
go test -v -run TestHandleNodeFailure ./internal/workflow/
DB_POSTGRES_DSN=... make test-functional       # -tags=functional ./tests/functional/... ; skips vacuously without the DSN
make e2e-local                                 # docker build + compose --profile e2e + go test -tags=e2e ./tests/e2e -timeout 15m
make run                                       # ./bin/fuse server on :9090
# NB: GET /v1/packages serves the *persisted* workflow.Package (packages.go:52 FindAll → ToPackageDTO),
# NOT the registry-side metadata. It echoes whatever transport string was stored, so it does NOT show
# the registry's hardcoded "internal" — only a Go test or a debugger can observe that.
curl -s localhost:9090/v1/packages | jq '.items[].functions[].metadata.transport'
curl -X POST "localhost:9090/v1/workflows/$WF/execs/$EXEC" \
  -H 'content-type: application/json' -d '{"result":{"status":"success","data":{}}}'  # the unvalidated callback
```

⚠️ There is no `make examples-ci` target and no `scripts/run-example-workflows.sh`; the real entry
points are `make seed` and `./bin/fuse seed examples --ci`. And there is no
`GET /v1/workflows/{workflowID}/status` route — it is `GET /v1/workflows/{workflowID}`, returning
`{workflowId, status}` only.

$ARGUMENTS
