# Phase 1 Data Model: AI Agent Node

These are **in-memory, per-execution** structures (no persistence, no schema migration). They
describe the node's input/output metadata and the runtime objects the reasoning loop manipulates.

## Node I/O metadata (`AgentFunctionMetadata`)

Mirrors `ChatFunctionMetadata()` in `chat.go`. `Transport = transport.Internal`,
`Input.CustomParameters = false`.

### Inputs

| Name | Type | Required | Default | Notes |
|------|------|----------|---------|-------|
| `input` | string | yes | — | The task / goal for the agent (FR-001, FR-015). |
| `provider` | string | no | engine default | Provider registry key (FR-001). |
| `model` | string | no | provider default | Model id. |
| `systemPrompt` | string | no | — | Optional system instruction. |
| `temperature` | float | no | provider default | Sampling temperature. |
| `maxIterations` | int | no | 10 | Clamped to `[1, 25]` (FR-007). |
| `allowedTools` | array | no | all eligible | Optional allowlist of function ids (FR-012). Falls back to comma-separated string if the graph validator rejects `array`. |

### Outputs

| Name | Type | Required | Notes |
|------|------|----------|-------|
| `output` | string | yes | Final text answer (FR-001, FR-014). |
| `usage` | map | no | Aggregated `{promptTokens, completionTokens, totalTokens}` across all steps (FR-011). |
| `steps` | array | no | Per-tool-call trace (FR-010). Each entry: `{tool, arguments, result|error}`. |

## Runtime entities

### Tool / ToolDescriptor
Derived from a registered function. Fields: `FunctionID` (real, e.g. `fuse/pkg/logic/sum`),
`MangledName` (`fuse__pkg__logic__sum`), `Description`, `Parameters` (JSON-Schema object built
from the function's `ParameterSchema`). Produced by `ToolRegistry.ListTools()`; converted to
`llm.Tool` for the request.

### Conversation (message slice)
Ordered `[]llm.Message` owned by the agent goroutine:
1. optional `system` (from `systemPrompt`),
2. `user` (the `input` task),
3. `assistant` replies (may carry `ToolCalls`),
4. `tool` results (`Role=tool`, `ToolCallID`, JSON content) — one per tool call.

### ToolCall
Model-issued request: `{ID, Name (mangled), Arguments (json.RawMessage)}`. Resolved to a real
function id via the mangled→real map; unknown/disallowed → error tool message (FR-009).

### Step / Trace Entry
`{tool: realFunctionID, arguments: map, result: map}` or `{tool, arguments, error: string}`.
Accumulated and emitted as `steps`.

### Usage Summary
Running totals of `promptTokens`/`completionTokens`/`totalTokens` summed across every
`ChatResponse.Usage` in the loop.

## Relationships

```
AgentNode --reads--> ToolRegistry --produces--> [ToolDescriptor] --maps to--> [llm.Tool]
   |                                                                   |
   | builds                                                            v
Conversation <---- appends ---- assistant reply (ToolCalls) ----> [ToolCall]
   |                                                                   | invoke (sync, in-process)
   | appends tool result                                               v
   +<------------------------------ FunctionResult.Output.Data <--- ToolRegistry.InvokeTool
   |
   +--> Step (trace) ; Usage (aggregated)
Final assistant message (no tool calls) --> output (+ usage + steps) --> Finish
```

## Validation rules

- `input` non-empty → else synchronous `FunctionError` (FR-015).
- `maxIterations` clamped to `[1, 25]` (FR-007).
- `provider` resolvable → else synchronous `FunctionError`.
- Each tool call: `Name` resolvable to an allowed real id, `Arguments` valid JSON → else recorded
  error tool message, loop continues (FR-009).
