---
name: function-package-authoring
description: Explains how an in-process FUSE node function is declared, registered, executed and tested — file layout, metadata, the FunctionResult contract including the async Finish/callback path, what ExecutionInfo actually carries, and the registration traps that silently produce a non-executable node. Use when adding or changing anything under internal/packages/functions/, when a node fails with "function X has no transport", or when a backlog task touches node dispatch.
---

# function-package-authoring

This pack covers the **in-process half** of node execution: a Go function compiled into the engine,
registered from code, dispatched to a `WorkflowFuncPool` worker.

The **out-of-process half** — a node backed by someone else's HTTP service, the invocation envelope,
ack semantics and the callback contract — is the remote node protocol. **It does not exist yet**:
**B-01** (write the spec) and **B-02** (implement the transport in Core) are both open backlog tasks,
and the inventory of what is actually shipped lives in the `remote-node-protocol` pack
(`.agents/skills/remote-node-protocol/SKILL.md`). Do not design a remote transport from this file;
point at that one. What this file gives B-02 is the seam it must plug into:
`transport.FunctionTransport`, `MapToRegistryPackage`'s branch, and the async completion route.

Backlog ids below are from the repo-root backlog (`BACKLOG_V2.md`). *If it has been renamed to
`BACKLOG.md`, the canonical file is the one titled "FUSE — product shape and backlog".*

## When to use

- Adding a function to an existing package, or adding a whole package.
- Diagnosing "function %s not found" / "function %s has no transport".
- Any task that reads or changes how a node's result reaches the workflow actor.

## Ground truth

| File | What it is authoritative for |
| --- | --- |
| `pkg/workflow/function.go` | The entire function contract: `type Function func(*ExecutionInfo) (FunctionResult, error)` |
| `pkg/workflow/execution_info.go` | Exactly what a function receives (5 fields, lines 14–23) |
| `pkg/workflow/fn_result.go` | The five result constructors — there are no others |
| `pkg/workflow/metadata.go` | The declarative metadata types an author fills in |
| `pkg/workflow/package.go` | `NewPackage` (L29), `NewFunction` (L69), `PackagedFunction.Function` is `json:"-"` (L24) |
| `internal/packages/loaded_package.go` | Registration-time mapping + the executable/metadata-only branch (L112–128) |
| `internal/packages/internal_packages.go` | `List()` (L41) — the one hardcoded slice of compiled-in packages |
| `internal/actors/workflow_func.go` | The worker that actually calls the function |
| `docs/adr/0024-package-registry-and-function-metadata.md` | Why the registry is metadata-described |

⚠️ **`.agents/rules/04-workflow-nodes.mdc` is wrong about this area and is `alwaysApply: true`.** Its
`FunctionMetadata` / `InputMetadata` / `OutputMetadata` samples invent fields that do not exist
(`FunctionMetadata.ID/Name/Description`, `InputMetadata.Required`, `OutputMetadata.Schema`) and omit
`Transport` and all edge metadata; the file is also duplicated verbatim (689 lines, body repeats at
L348). Trust `pkg/workflow/metadata.go`, not the rule.

## The contract, in one screen

```go
// pkg/workflow/function.go
type Function func(*ExecutionInfo) (FunctionResult, error)
```

`ExecutionInfo` — `pkg/workflow/execution_info.go:14-23`. **This is all a function gets today:**

| Field | Type | Notes |
| --- | --- | --- |
| `WorkflowID` | `workflow.ID` | run id |
| `ExecID` | `workflow.ExecID` | UUIDv8 with the thread id in bytes 6–7; `ExecID.Thread()` recovers it |
| `Environment` | `string` | ADR-0031 resolution scope; scope data, not a secret |
| `Input` | `*FunctionInput` | typed accessors over the mapped input KV |
| `Finish` | `func(FunctionOutput)` | **nil until a transport binds it** (`internal/packages/transport/internal.go:47`) |

There is **no `context.Context`**, no cancellation, no logger, and no graph node id.
`internal/actors/workflow_func.go:96` is literally `_ = nodeCtx` — the node span context is created
and discarded. **A-04** owns adding the context; **B-02** depends on it (a remote call cannot inject
`traceparent` without one). See the `observability-tracing` pack.

## File layout — copy an existing package

Every package is one directory `internal/packages/functions/<pkg>/` with `package.go` plus one file
per function named after the function. Confirmed for `debug`, `logic`, `http` and `system`. `ai`
follows it for its two *registered* functions (`chat.go`, `agent.go` — `package.go` registers only
`ChatFunctionID` and `AgentFunctionID`) and additionally carries helper files that are not functions
(`context.go`, `structured.go`, `tools.go`, `usage.go`).

```
internal/packages/functions/logic/
  package.go   // doc comment + `const PackageID` + `New() *workflow.Package`
  sum.go       // SumFunctionID + SumFunctionMetadata() + SumFunction()
  sum_test.go
```

Each function file exports exactly three things:

```go
const SumFunctionID = "sum"                                  // no slash, ever
func SumFunctionMetadata() workflow.FunctionMetadata { ... }  // a func, never a package-level var
func SumFunction(execInfo *workflow.ExecutionInfo) (workflow.FunctionResult, error) { ... }
```

Worked end to end: `internal/packages/functions/logic/sum.go` (57 lines) — metadata declares one
required input `values []float64` and one output `sum`; the body reads
`execInfo.Input.GetFloat64SliceOrDefault("values", ...)` and returns
`workflow.NewFunctionResult(workflow.FunctionSuccess, map[string]any{"sum": sum})`.

Package ids are namespaced `fuse/pkg/<domain>` (`fuse/pkg/http`, `fuse/pkg/logic`, `fuse/pkg/debug`,
`fuse/pkg/ai`). `system` is the one deliberate exception — `internal/packages/functions/system/package.go:9`
sets `PackageID = "system"` because its functions are engine-intercepted. Do not "fix" it; the
package id is derived by splitting the schema's `function` string at the **last** slash
(`internal/services/graph_service.go:314`), so every existing schema depends on it.

## Registration: two paths, only one produces an executable node

| Path | Entry point | Result |
| --- | --- | --- |
| **Code-backed internal** | `internal_packages.go:List()` → `PackageService.RegisterInternalPackages` → `Registry.Register` → `MapToRegistryPackage` | `NewLoadedInternalFunction` wires `InternalFunctionTransport`; the node runs |
| **API / persistence** | `PUT /v1/packages/{packageID:.+}` → `dtos.FromPackageDTO` → `PackageService.Save` → `Registry.Register` | metadata-only, `Transport == nil`; dispatch errors with `function %s has no transport` (`loaded_package.go:41`) |

The branch is `internal/packages/loaded_package.go:112`:
`if function.Metadata.Transport == transport.Internal && function.Function != nil`. Anything else
falls to the metadata-only `NewLoadedFunction` at L124. **B-02** is the task that makes the else-branch
produce a real remote transport.

Adding a package is a **3-file minimum** change:

1. the new `internal/packages/functions/<pkg>/` directory;
2. append it to `internal/packages/internal_packages.go:List()` (L41) — **omit this and the package does
   not exist at runtime**, with no error anywhere;
3. only if it needs dependencies: widen `NewInternal` (L25) and the providers in
   `internal/app/di/di.go` `PackageModule` (which provides `NewPackageRegistry`, `NewInternal`,
   `providePackageRegistration`).

Dependencies use a closure factory: `makeChatFunction(providers, usage) workflow.Function`
(`ai/chat.go:94`), with the ports declared *inside* `functions/<pkg>` and adapted in
`internal/packages` (`usage_recorder.go`, `agent_tools.go`) so `functions/<pkg>` never imports
`internal/packages` — that would be an import cycle.

## Metadata and schemas

`workflow.FunctionMetadata{Transport, Input, Output}`. Slice-shaped here; converted to the map-shaped
`internal/packages.FunctionMetadata` at registration.

- `Transport: transport.Internal` — imported from `github.com/open-source-cloud/fuse/internal/packages/transport`.
- `Input.Parameters []ParameterSchema` — `{Name, Type, Required, Validations, Description, Default}`.
- `Input.CustomParameters: true` means schemaless input read straight from `Input.Raw()` (see `logic/if.go`).
- `Output.ConditionalOutput` + `ConditionalOutputField` + named `Output.Edges` with
  `ConditionalEdge.Value` drive conditional routing (`logic/if.go` declares `if-true` / `if-false`).
- Empty slices are written `make([]T, 0)`, not nil — dominant convention (`sum.go:29`
  `ParameterSchema`, `sum.go:42` `OutputEdgeMetadata`, `timer.go:33,37-38`). `system/sleep.go`,
  `wait.go` and `subworkflow.go` omit `Output.Edges` entirely, while `system/foreach.go:42` declares
  two; both shapes compile.

## FunctionResult: the five constructors, and only five

| Constructor | `Async` | Use |
| --- | --- | --- |
| `NewFunctionResult(status, data)` | false | general sync result |
| `NewFunctionResultSuccess()` | false | success, nil data |
| `NewFunctionResultSuccessWith(data)` | false | success with data |
| `NewFunctionResultError(err)` | false | **returns `(result, nil)`** — logical failure is a `FunctionError` result, not a Go error |
| `NewFunctionResultAsync()` | true | "I will call `Finish` later" |

There is no `NewFunctionResultAsyncWith`. The Go `error` return is reserved for *unexpected* execution
failures; `workflow_func.go:141-153` rewrites a non-nil error into a `FunctionError` result anyway and
logs a warning when you return both, so returning both is redundant.

## The async path

**In-process (TODAY, works):** return `NewFunctionResultAsync()` synchronously *and* spawn a goroutine
that eventually calls `execInfo.Finish(...)`. Smallest reference: `logic/timer.go:44-70` (70 lines).
Richest reference: `ai/chat.go:94-174` (`makeChatFunction`) — every error path calls
`execInfo.Finish(workflow.NewFunctionOutput(workflow.FunctionError, map[string]any{"error": err.Error()}))`
before returning.

`Finish` is bound by `internal/packages/transport/internal.go:47`; it sends
`messaging.NewAsyncFunctionResultMessage` to the handler addressed by
`gen.Atom(actornames.WorkflowHandlerName(wfID))` — by **name, not PID** (see the comment at L12-17;
sending by captured PID from a pool worker has been observed to fail with `gen.ErrUnsupported`).
It lands at `workflow_handler.go:handleMsgAsyncFunctionResult` (L313).

**External callback (TODAY, exists, unvalidated) — described in ONE place, and it is not this file.**
The route is `POST /v1/workflows/{workflowID}/execs/{execID}`
(`internal/actors/mux_worker.go:79`), handled by `internal/handlers/async_function_result.go`. It is
the seam B-02 builds on, and it validates nothing today. Do **not** restate its contract from here:
the body DTO, the exact missing checks, the duplicate-callback consequence and the fix template all
live in `.agents/skills/remote-node-protocol/SKILL.md` § TODAY, which is the single source for
anything crossing the engine↔worker boundary. This pack owns only the **in-process** `Finish` path
above.

**Why async is the default for anything slow (A-10).** `internal/actors/workflow_func_pool.go:40` is
`act.PoolOptions{PoolSize: 3}` — three workers **per workflow instance**, hardcoded. A synchronous node
holds one for its whole duration, so one 90-second call caps that run at two other concurrent branches.
`fuse/pkg/http/request` (`internal/packages/functions/http/request.go:128`) is synchronous and blocks
this way today. A-10 makes the size configurable and requires the remote transport never to hold a
worker while awaiting a callback.

## How a schema references a function

`examples/workflows/github-request-example.json` is the worked example. A node names the **full** id:

```json
{ "id": "github-http-request", "function": "fuse/pkg/http/request" }
```

Edge `input` entries feed the declared parameters and read the declared outputs:

- `{"source": "schema", "value": "https://api.github.com", "mapTo": "host"}` — literal from the schema.
- `{"source": "flow", "variable": "github-http-request.status", "mapTo": "statusCode"}` — a prior node's
  output, addressed **by node id**. (Addressing by execution is **A-11**, and it is not built.)

Binding happens at upsert time in `internal/services/graph_service.go:populateNodeMetadata` (L295):
split at the last slash → `Registry.Get(pkgID)` → `LoadedPackage.GetFunctionMetadata(node.Function)`
with the **full** id → `graph.UpdateNodeMetadata`. A schema naming an unregistered function fails
`PUT /v1/schemas/{schemaID}` and fails `fuse seed examples`.

If a new example cannot run in CI, extend `shouldSkipExampleWorkflow`
(`internal/app/cli/seed.go:184`) — it currently skips `github-request-example.json` and any file whose
JSON contains `fuse/pkg/logic/timer` or `fuse/pkg/ai/`. Give a new example a **unique schema id**: files
are seeded in sorted-name order and the last file wins for a duplicated id.

## Tests

| Kind | Template | Package |
| --- | --- | --- |
| Function behaviour | `internal/packages/functions/logic/sum_test.go` | external (`package logic_test`); hand-builds `&workflow.ExecutionInfo{WorkflowID, ExecID, Input, Finish: nil}` |
| Metadata / package shape | `internal/packages/functions/system/package_test.go` | same package; asserts function count, parameter names, `Required`, and the `Full*FunctionID` constants |
| Registry / transport | `internal/packages/loaded_package_test.go` | `TestExecuteFunction_PersistenceDecodedInternalFunc_DoesNotPanic` (L33), `TestExecuteFunction_CodeBackedInternalFunc_Runs` (L60), `TestRegister_DataOnlyCopy_DoesNotDowngradeExecutableFunc` (L83) — uses `Encode`/`DecodePackage` to simulate a persistence round-trip |

⚠️ `internal/packages/functions/http/` contains **only** `package.go` and `request.go` — zero test files.
Anything B-02 asserts about HTTP nodes is new test surface, including the synchronous baseline.

## Traps (all confirmed in this tree)

| Trap | Evidence |
| --- | --- |
| **Two packages named `transport`.** Metadata must import `internal/packages/transport` (defines `Internal`). `pkg/transport` defines only `HTTP`/`gRPC` — and `gRPC` is lowercase, so unexported and unusable outside that package. Importing the wrong one compiles into a metadata-only registration that silently never executes. | `pkg/transport/type.go` (11 lines); `internal/packages/transport/type.go` |
| **`Functions` is keyed by the FULL id.** `GetFunctionMetadata` and `ExecuteFunction` both expect `"fuse/pkg/http/request"` despite the parameter being named `functionID`. Passing `"request"` returns `function request not found`. | `loaded_package.go:66`, `:26`, `:35` |
| **A function id must not contain a slash.** `graph_service.go:314` splits at the last slash, so `fuse/pkg/http/request_async` resolves to package `fuse/pkg/http`, function `request_async`. | `graph_service.go:314` |
| **`PackagedFunction.Function` is `json:"-"`.** Any package arriving as JSON (API, Postgres reload, replication) loses its code pointer. `registry.go:47-65` exists solely to stop that downgrading an executable registration — it preserves the whole previously-registered `*LoadedFunction`. Do not "simplify" it. | commit 802675d |
| **The REST package DTOs drop all edge metadata**, both directions. `internal/dtos/package.go` has no `Edges` field anywhere, so `GET /v1/packages` shows `fuse/pkg/logic/if` without its `if-true`/`if-false` edges, and an API-registered package cannot declare conditional routing. It bites in-process authoring only when a package round-trips through the API; the registration-loss inventory and its backlog routing are owned by `.agents/skills/remote-node-protocol/SKILL.md`. | grep `Edges` in `internal/dtos/package.go` → no hits |
| **Adding a sync, non-`CustomParameters` function silently exposes it to every `ai/agent` as an LLM tool.** If it has side effects, add its full id to `interceptedOrAsyncFunctionIDs` in the same PR. | `agent_tools.go:18-24`, `isExposableTool` L95-111 |
| **The registry is a process-global singleton** (`var pkgRegistry Registry`, `registry.go:28`); `NewPackageRegistry` returns the same instance every call. Tests that register leak across tests in the same binary — use `MapToRegistryPackage` directly, as `loaded_package_test.go` does. | `registry.go:28-38` |
| **`FunctionMetadata.Concurrency` and `.RateLimit` are never populated.** `MapToRegistryPackage` does not set them and `pkg/workflow.FunctionMetadata` has no such fields, so the guards at `workflow_func.go:102`/`:108` are unreachable for code-backed packages. **No backlog task owns this** — file it, don't fix it in passing. | `internal/packages/function_metadata.go:14-15` vs `loaded_package.go:63-131` |
| **Intercepted `system/*` functions never reach a worker** — `handleWorkflowRunFunctionAction` (L671) switches on `SleepFullFunctionID`/`WaitFullFunctionID`/`SubWorkflowFullFunctionID`/`ForEachFullFunctionID` at L675-688 and returns. Their bodies are placeholders that return success (`system/wait.go:32`). Their **replay** is broken and is **A-02**. | `workflow_handler.go:675-688` |
| **ForEach bypasses that interception.** `spawnForEachBatch` builds an `ExecuteFunctionMessage` and sends straight to the pool (`workflow_handler.go:1043-1046`), so `system/*` inside a ForEach body hits the no-op placeholder and silently does nothing. Owned by **C-08**. | `workflow_handler.go:1013-1048` |
| **`http.ErrMethodNotAllowed` is declared and never used** (grep: two hits, both the declaration); the method is not checked against an allowlist. | `request.go:22-23` |

## Commands (confirmed to exist)

```bash
make lint && make build && make test            # the mandated gate, in this order
go test -v ./internal/packages/...              # registry, mapping, agent-tool exposure
go test -v -run TestSumFunction ./internal/packages/functions/logic/
go test -v ./pkg/workflow/...                   # input / result / exec-id contract
make swagger                                    # REQUIRED on a fresh clone before build/test
make build && ./bin/fuse seed examples -l debug # `make seed`; upserts examples/workflows/*.json
./bin/fuse seed examples --ci                   # applies shouldSkipExampleWorkflow
make run                                        # ./bin/fuse server on :9090
curl -s http://localhost:9090/v1/packages | jq '.items[].id'   # dtos.PackageListResponse{metadata, items}
curl -X POST "http://localhost:9090/v1/workflows/$WF/execs/$EXEC" \
  -H 'Content-Type: application/json' -d '{"result":{"status":"success","data":{}}}'
```

Note: there is no `make examples-ci` target and no `scripts/run-example-workflows.sh` in this tree.
The real entry points are `make seed` and the `--ci` flag.

$ARGUMENTS
