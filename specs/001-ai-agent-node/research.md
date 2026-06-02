# Phase 0 Research: AI Agent Node

All decisions below were validated against the current code on branch `main`. They are the
substance of the implementation and resolve the unknowns the agent loop introduces.

## R1 — How does the agent invoke other functions in-process?

**Decision**: The agent receives its tool capability as an **injected port** (`ai.ToolRegistry`,
provided at `makeAgentFunction` construction). Synchronous tools run through a **handle-free**
path: `ToolRegistry.InvokeTool(functionID, execInfo)` → `LoadedPackage.ExecuteFunctionSync` →
`InternalFunctionTransport.ExecuteSync`, which executes the function inline with **no actor
handle**. `workflow.ExecutionInfo` stays a clean input/output + completion contract — it gains
**no** `Handle` field, and the `ai` node package imports no actor/runtime types.

**Rationale**: The `workflow.Function` signature is `func(*ExecutionInfo) (FunctionResult, error)`;
sub-invocation is a capability of a *distinct, function-orchestrated* node class (the agent), not
of the universal execution context. Every surveyed engine (n8n, Temporal, LangGraph, Prefect,
Dagster, Airflow, Camunda) injects this capability into the orchestrating component rather than
exposing it on a leaf step's run-context. And because Phase B exposes only synchronous tools —
which return inline and never fire `Finish` — the worker actor handle is genuinely unnecessary
here.

**Superseded approach**: an earlier iteration added `ExecutionInfo.Handle any`, populated by the
transport. It was removed: the field leaked runtime access onto the shared contract, was untyped,
and was never actually dereferenced for sync tools.

**Deferred (Phase C)**: async tool invocation needs the per-execution worker handle plus a
correlation channel. That is a *separate, typed* per-execution runtime port injected into
orchestrating nodes — never a field on `ExecutionInfo`. See ADR-0027.

## R2 — How does `ai` enumerate/invoke registry functions without an import cycle?

**Decision**: Declare a port interface `ToolRegistry` (and `ToolDescriptor`) **in the `ai`
package**; implement it with an adapter that lives in package `packages`
(`internal/packages/agent_tools.go`) and is injected into `ai.New`.

**Rationale**: `packages` imports `ai` today (`internal_packages.go` calls `ai.New`). Therefore
`ai` must not import `packages`. `ai` *can* import `internal/actors/actor` (it only imports
`ergo/gen` — no cycle). The adapter, being in `packages`, freely uses `LoadedPackage`, the
`Registry`, and the `system.*FullFunctionID` constants, and depends on `ai` only for the port
types — preserving the existing dependency direction.

**Alternatives considered**: Move the registry types to a neutral package — larger refactor.
Reflection-based discovery — fragile, untyped.

## R3 — How are synchronous tool results read back inline?

**Decision**: A synchronous internal function returns `workflow.FunctionResult{Async:false,
Output:{Status, Data}}`; the agent reads `result.Output.Data` immediately. The handle-free
`ExecuteSync` path (R1) runs the function with no handle and binds a guard `Finish` that
logs-and-ignores any unexpected async-completion attempt. Any tool returning `Async:true` is
treated as "unsupported" feedback (not awaited).

**Rationale**: `InternalFunctionTransport.Execute` rebinds `Finish` to send an
`AsyncFunctionResultMessage` to the WorkflowHandler. For an async tool that result would be routed
to the handler, *not* back to the agent goroutine — hence Phase B's sync-only constraint. Sync
tools never call the rebound `Finish`, so their data is solely in the returned `FunctionResult`.

## R4 — Which functions are eligible as tools (exclusion predicate)?

**Decision**: Expose a function iff **all** hold:
1. `Transport == transport.Internal`,
2. `Input.CustomParameters == false` (excludes schemaless functions like `logic/if`),
3. full function id ∉ denylist: `system/sleep`, `system/wait`, `system/subworkflow`,
   `system/foreach` (intercepted by the WorkflowHandler), `fuse/pkg/logic/timer` (async),
   `fuse/pkg/ai/chat`, `fuse/pkg/ai/agent` (async; no agent-in-agent in Phase B).

**Rationale**: Intercepted/async functions route their results away from the agent goroutine;
schemaless functions can't be described to the model as a JSON-Schema tool. Use the exported
`system.*FullFunctionID` constants rather than string literals to stay in sync.

## R5 — `ParameterSchema` → JSON Schema

**Decision**: Build `map[string]any` = `{"type":"object","properties":{<name>:{"type":<json>,
"description":<desc>}}, "required":[<names where Required>]}`. Type map: `string→string`,
`int→integer`, `float→number`, `bool→boolean`, `map→object`, `array`/`slice→array`,
default→`string`. Include `"default"` when `Default != nil`. Empty params → `{"type":"object",
"properties":{}}`.

**Rationale**: `llm.Tool.Parameters` is `map[string]any` (a JSON-Schema object); `ToolCall.
Arguments` is `json.RawMessage`. Validation parsing (min/max/enum) is deferred — type + required +
description suffices for Phase B.

## R6 — Tool-name mangling

**Decision**: `mangle = ReplaceAll(id, "/", "__")`; `demangle = ReplaceAll(name, "__", "/")`.
Build a `map[mangledName]realID` once per run for O(1) reverse lookup; unknown names from the
model produce an error tool-result message rather than crashing the goroutine.

**Rationale**: Provider tool-name constraints disallow `/`; `__` round-trips safely for the known
function-id shapes (`system/sleep`, `fuse/pkg/logic/sum`).

## R7 — Loop termination & exactly-once `Finish`

**Decision**: Loop bounded by `maxIterations` (default 10, hard cap 25) and wrapped in
`context.WithTimeout(agentTimeout = 5*time.Minute)`. Terminate on: no tool calls / stop finish
reason → success with `output`+`usage`+`steps`; iteration or time limit → **error** terminal
output; provider error → error output. `Finish` is called **exactly once** on every path (mirrors
`chat.go` discipline). Missing `input`, unknown provider, or nil `Handle` fail **synchronously**
(before the goroutine) via `NewFunctionResultError`.

**Rationale**: Guarantees SC-003 (every run terminates) and FR-013 (deterministic single result),
and keeps the worker pool free (FR-006 / SC-004).

## R8 — DI ordering (adapter vs. registry population)

**Decision**: `NewInternal(providers, registry)` builds the adapter holding the `Registry`
*interface* (not a snapshot). `ListTools()` reads it lazily at agent-execution time, after
`RegisterInternalPackages` has populated it. fx auto-injects `Registry` (already provided at
`di.go:46`), so no `di.go` change is needed.

**Rationale**: Construction order (constructor runs before registration) is safe because the
adapter never reads the registry at construction time.
