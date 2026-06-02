# 0007. Agent reasoning loop & tools-from-functions

- Status: Accepted
- Date: 2026-06-01
- Deciders: FUSE maintainers

## Context and Problem Statement

Phase B of the agent roadmap ([ADR-0005](0005-ai-agents-as-workflow-nodes-phased-roadmap.md))
adds an `ai/agent` node that reasons in a loop and calls **tools** to accomplish a
task. FUSE already has a registry of package functions with typed parameter
metadata. The questions: where does the reasoning loop run without blocking the
engine, how do existing functions become LLM tools, and how does the agent invoke
them and feed results back?

## Decision Drivers

- Don't block the `WorkflowFunc` worker pool — an agent loop is long-running.
- Reuse existing functions as tools rather than defining a parallel tool system.
- Stay within the deterministic node model (the agent is one node; its result flows
  on via normal edges).
- Be honest about which tools are safe to expose in the first iteration.

## Considered Options

- **Run the loop inside the `ai/agent` function as a goroutine, completing via the
  existing async `Finish` mechanism; tools = package functions invoked in-process.**
- **Run the loop synchronously inside the worker** (occupies a pool slot for the
  whole multi-step interaction).
- **Model each tool call as its own graph node / sub-execution** routed through the
  actor system.

## Decision Outcome

Chosen option: **async in-function loop with functions-as-tools.** The `ai/agent`
function returns `NewFunctionResultAsync()` and runs its loop in a goroutine,
reporting the final answer via `execInfo.Finish(...)` — the same mechanism
`logic/timer` already uses — so the worker pool is freed immediately. Tools are
discovered from the package registry; each function's
`FunctionMetadata.Input.Parameters` is converted to a JSON-Schema `llm.Tool`. When
the model requests a tool, the agent invokes the function **in-process** via
`LoadedPackage.ExecuteFunction` and feeds the result back as a tool message,
looping until a final answer or `maxIterations`.

Constraints for Phase B:

- **Only synchronous tools are exposed.** Async/intercepted functions
  (`system/sleep`, `system/wait`, `system/subworkflow`, `system/foreach`,
  `logic/timer`) route their result to the `WorkflowHandler`, not back to the agent
  goroutine, so they cannot be tools yet. Schemaless `CustomParameters` functions
  (e.g. `logic/if`) are also excluded from auto-exposure.
- **Tool-name mangling**: tool names use `package__function` (`/`↔`__`) to satisfy
  provider name constraints, mapped back to the real function ID on invocation.
- **`ExecutionInfo` gains a `Handle`** so the agent goroutine can call
  `ExecuteFunction` (the goroutine uses `handle.Node()`, never `handle.Send`).

### Consequences

- Good: agents reuse the entire function catalog as tools, with schemas for free.
- Good: the worker pool stays free during long agent interactions; errors surface as a
  `FunctionError` output and flow through normal error handling.
- Bad: a real (but documented) limitation — async tools are unavailable until Phase C,
  which adds a sub-execution correlation channel.
- Neutral: tool selection/validation happens at agent-execution time against the registry.

## Pros and Cons of the Options

### Async in-function loop + functions-as-tools (chosen)

- Good: non-blocking; reuses functions, schemas, and the proven async pattern; stays a
  single node.
- Bad: async-tool support deferred; needs the `ExecutionInfo.Handle` addition.

### Synchronous in-worker loop

- Good: simplest to write.
- Bad: holds a pool slot for the entire multi-step interaction; starves throughput.

### Tool calls as graph nodes / sub-executions

- Good: uniform with the rest of the engine; could support async tools immediately.
- Bad: results route to the `WorkflowHandler`, not the agent; needs a correlation/reply
  channel that doesn't exist yet — significant new machinery for Phase B.

## More Information

- Code: `internal/packages/functions/ai/agent.go` (reasoning loop + metadata) and
  `internal/packages/functions/ai/tools.go` (the `ToolRegistry` seam, `package__function`
  name mangling, and `ParameterSchema`→JSON-Schema conversion); the registry adapter and
  exclusion predicate live in `internal/packages/agent_tools.go`. The worker handle reaches
  the agent goroutine via `workflow.ExecutionInfo.Handle`, populated at the single choke point
  `internal/packages/transport/internal.go`. Tool invocation goes through
  `internal/packages/loaded_package.go` (`ExecuteFunction`); precedent for the async pattern is
  `internal/packages/functions/logic/timer.go`.
- Specified and delivered through the spec-driven flow under `specs/001-ai-agent-node/`.
- Accepted: Phase B shipped — the `ai/agent` node exposes synchronous, declared-parameter
  functions as tools. The async-tool limitation is lifted by a future sub-execution correlation
  channel (see [ADR-0027](0027-async-tool-invocation-sub-execution-channel.md)); streaming and a
  native Anthropic provider remain Phase C.
- Related: [ADR-0005](0005-ai-agents-as-workflow-nodes-phased-roadmap.md),
  [ADR-0006](0006-llm-provider-abstraction-and-multi-provider-strategy.md).
