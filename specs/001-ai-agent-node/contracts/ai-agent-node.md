# Contract: `ai/agent` Node

**No new HTTP endpoint.** The agent is a workflow node (function id `fuse/pkg/ai/agent`,
`transport.Internal`) reached through the existing API: a workflow whose graph contains an agent
node is triggered via `POST /v1/workflows/trigger`, and the run is inspected via
`GET /v1/workflows/{workflowID}/status`. This contract specifies the node's behaviour.

## Function identity
- Package: `fuse/pkg/ai`
- Function: `agent` → full id `fuse/pkg/ai/agent`
- Transport: `Internal`; completion: **async** (`NewFunctionResultAsync()` then `Finish`).

## Request (node inputs)
See [data-model.md](data-model.md#inputs). Minimum: `{ "input": "<task>" }`.

## Response (node outputs)
On success (`FunctionSuccess`):
```json
{
  "output": "<final answer text>",
  "usage": { "promptTokens": 0, "completionTokens": 0, "totalTokens": 0 },
  "steps": [ { "tool": "fuse/pkg/logic/sum", "arguments": { "a": 2, "b": 3 }, "result": { "sum": 5 } } ]
}
```
On terminal failure (`FunctionError`): `{ "error": "<reason>" }` — e.g. missing input, unknown
provider, provider failure, `max iterations reached`, time limit exceeded. Flows through normal
error handling (FR-013, FR-014).

## Behavioural contract
1. **Tool derivation**: tools = eligible registry functions (see exclusion predicate in
   [research.md R4](research.md)), optionally ∩ `allowedTools`.
2. **Loop**: call model with `Tools` + `ToolChoice:"auto"`; execute requested tools in-process;
   append results as `tool` messages; repeat.
3. **Termination**: final answer (no tool calls / stop), OR `maxIterations` reached → error, OR
   `agentTimeout` exceeded → error. Exactly one `Finish`.
4. **Resilience**: unknown/disallowed tool or tool error → recorded, fed back to the model, loop
   continues (does not abort the run).
5. **Non-blocking**: returns async immediately; the `WorkflowFunc` worker is freed for the whole
   interaction.

## Error modes (all → single terminal result)
| Condition | When | Result |
|-----------|------|--------|
| Missing `input` | before loop (sync) | `FunctionError{error:"...input is required"}` |
| Unknown provider | before loop (sync) | `FunctionError` |
| Missing/invalid `Handle` | before loop (sync) | `FunctionError` |
| Provider/model call fails | in loop | `FunctionError` via `Finish` |
| `maxIterations` reached | in loop | `FunctionError{error:"max iterations reached"}` |
| `agentTimeout` exceeded | in loop | `FunctionError` |

## Test contract (no live model)
A scripted stub `llm.Provider` returns a tool-call on call #1 and a final answer on call #2; a
fake `ToolRegistry` returns a sync `FunctionResult`; a fake `toolHandle` satisfies the assertion.
Assertions: final `output`, the `tool` message threaded with the matching `ToolCallID`, aggregated
`usage`, recorded `steps`, and bounded termination.
