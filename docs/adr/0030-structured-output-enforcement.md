# 0030. Structured / JSON output enforcement for `ai/chat` & `ai/agent`

- Status: Accepted (tool-forced "respond" path shipped; provider-native `response_format` deferred)
- Date: 2026-06-02
- Deciders: FUSE maintainers

## Context and Problem Statement

The `ai/chat` and `ai/agent` nodes return free-form text in their `output`
([ADR-0007](0007-agent-reasoning-loop-and-tools-from-functions.md)). Downstream consumers —
edge-based input mapping ([ADR-0013](0013-workflow-schema-model-and-input-mapping.md)) and
conditional routing with expr-lang ([ADR-0015](0015-conditional-routing-with-expr-lang.md)) — work
far better with **typed, structured** data than with prose an author must parse. Modern models
expose structured-output features (JSON-schema response formats, tool-forced output), but support
differs across providers (the OpenAI-compatible path vs a future native Anthropic provider). We
need a decision on how authors declare an expected output shape and how the engine enforces it.

## Decision Drivers

- Make agent/chat output directly consumable by mapping
  ([ADR-0013](0013-workflow-schema-model-and-input-mapping.md)) and conditions
  ([ADR-0015](0015-conditional-routing-with-expr-lang.md)).
- Work across providers despite uneven structured-output support
  ([ADR-0006](0006-llm-provider-abstraction-and-multi-provider-strategy.md)).
- Reuse existing schema machinery — `ParameterSchema` and the
  `ParameterSchemaToJSONSchema` converter already built in
  `internal/packages/functions/ai/tools.go` — and validation (go-playground/validator).
- Keep it optional and backward compatible (text output remains the default).

## Considered Options

- **A — No enforcement (status quo).** Authors parse text downstream.
- **B — Provider structured output.** An optional `outputSchema` input drives the provider's
  native structured-output / `response_format` feature where supported; the engine validates the
  result and exposes the parsed object.
- **C — Tool-forced output.** Define a synthetic "respond" tool whose parameters are the output
  schema; the model "calls" it to deliver the final answer. Works on any provider that supports
  tool calling (which the agent already requires).

## Decision Outcome

**Proposed — leaning to Option B with Option C as the cross-provider fallback, deferred.** Where a
provider supports native structured output, use it (B); otherwise fall back to the tool-forced
"respond" pattern (C), which the agent's existing tool loop already makes natural. Both reuse
`ParameterSchemaToJSONSchema` for the schema and validate the parsed result before exposing it as
`output`. Deferred pending a per-provider capability mapping in the `llm.Provider` layer.

### Consequences

- Good: typed agent/chat output that flows cleanly into mapping and conditions.
- Good: reuses the schema converter shipped for Phase B tools.
- Bad: needs a per-provider capability map and a validation/repair loop for malformed output.
- Neutral: free-form text stays the default; this is opt-in per node.

## Pros and Cons of the Options

### A — No enforcement
- Good: nothing to build.
- Bad: brittle downstream parsing; poor fit for conditions/mapping.

### B — Provider structured output (chosen lean)
- Good: best fidelity where supported.
- Bad: uneven provider support; needs capability detection.

### C — Tool-forced "respond" (chosen fallback)
- Good: provider-agnostic; reuses the tool loop.
- Bad: slightly indirect; relies on the model choosing the respond tool.

## More Information

- **Shipped (Option C, tool-forced — universal)**: `ai/chat` and `ai/agent` accept an optional
  `outputSchema` input (a list of `{name,type,required,description}` ParameterSchema). When set, the
  node offers a single forced `respond` tool whose parameters are the JSON Schema
  (`ParameterSchemaToJSONSchema`) with `ToolChoice: "required"`, parses the tool-call arguments, and
  validates them against the schema; one bounded repair retry on invalid output, then a
  `FunctionError`. The validated object becomes the node's `output` (instead of free-form text); the
  agent runs its normal loop, then coerces the final answer. Absent `outputSchema` = unchanged.
  Works on every provider (all support tool calling). Logic in
  `internal/packages/functions/ai/structured.go`. **Deferred (Option B optimization)**:
  provider-native `response_format` / `json_schema` with a per-provider capability map for providers
  that support it natively.
- Reuse: `internal/packages/functions/ai/tools.go` (`ParameterSchemaToJSONSchema`),
  `pkg/workflow/metadata.go` (`ParameterSchema`).
- Related: [ADR-0006](0006-llm-provider-abstraction-and-multi-provider-strategy.md),
  [ADR-0007](0007-agent-reasoning-loop-and-tools-from-functions.md),
  [ADR-0013](0013-workflow-schema-model-and-input-mapping.md),
  [ADR-0015](0015-conditional-routing-with-expr-lang.md).
