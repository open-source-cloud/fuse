# 0013. Workflow schema model and edge-based input mapping

- Status: Accepted
- Date: 2026-06-01
- Deciders: FUSE maintainers

> Backfill ADR: records a foundational decision already implemented.

## Context and Problem Statement

A workflow must be defined declaratively (so it can be stored, versioned, and triggered) and
compiled into something executable. We need a definition format for the graph and a way to move
data between nodes whose input/output shapes don't have to match exactly.

## Decision Drivers

- Declarative, JSON/YAML-serializable definition that validates before execution.
- Decouple a node's output shape from the next node's input shape.
- Support both static (literal) inputs and data flowing from prior nodes.
- Compile cleanly to the runtime graph + thread model
  ([ADR-0011](0011-threading-model-and-foreach.md)).

## Considered Options

- **Three-layer schema (Graph/Node/Edge) with per-edge input mappings** (`schema`/`flow` sources).
- **Nodes wire directly to each other** with implicit pass-through of full outputs.
- **A single flat document** without an explicit edge object.

## Decision Outcome

Chosen option: **a composed three-layer schema with input mapping carried on edges.**

- `GraphSchema` — `ID`, `Name`, `Nodes`, `Edges`, optional `Metadata`/`Tags`, `Timeout`,
  `Concurrency`, `TriggerConfig` (`internal/workflow/graph_schema.go`).
- `NodeSchema` — `ID`, `Function` (`package/function`), optional `Retry`, `Timeout`, `Merge`
  (`node_schema.go`).
- `EdgeSchema` — `ID`, `From`, `To`, optional `Conditional`, `Input []InputMapping`, `OnError`
  (`edge_schema.go`).

Schemas are submitted via `PUT /v1/schemas/{schemaID}`, validated with `go-playground/validator`,
and compiled by `internal/workflow/graph.go` into a runtime `Graph` (node/edge lookup maps, the
first node as trigger, thread assignment).

**Input mapping** lives on each edge. An `InputMapping{Source, Variable, Value, MapTo}` has two
sources: `SourceSchema` (a static literal `Value`) and `SourceFlow` (pull `Variable` =
`"nodeID.outputField"` from the workflow's accumulated `aggregatedOutput` store). At edge
traversal, values are type-coerced (`typeschema.ParseValue`) and validated against the target
node's parameter schema, then handed to the downstream node as its input. Outputs are recorded
into `aggregatedOutput` keyed by node id after each step, which is what `SourceFlow` reads.

### Consequences

- Good: definitions are storable/validatable/versionable
  ([ADR-0014](0014-schema-versioning-and-rollback.md)); nodes compose without rigid contracts.
- Good: explicit edges are the natural home for conditions ([ADR-0015](0015-conditional-routing-with-expr-lang.md)),
  error routing (`OnError`), and merge wiring.
- Bad: input mapping is an extra concept authors must learn; mapping/coercion errors surface at
  runtime, not at schema-validation time.
- Neutral: `Metadata`/`Tags` are open maps reserved for tooling.

## Pros and Cons of the Options

### Three-layer schema + edge input mapping (chosen)

- Good: clean separation; flexible data binding; edges carry routing/error/merge semantics.
- Bad: more concepts; runtime mapping errors.

### Direct node-to-node pass-through

- Good: less to author for simple flows.
- Bad: forces output/input shapes to match; no clean place for conditions/error routing.

### Flat document without edges

- Good: compact.
- Bad: routing, conditions, and data binding become implicit and hard to validate/extend.

## More Information

- Code: `internal/workflow/graph_schema.go`, `node_schema.go`, `edge_schema.go`, `graph.go`;
  mapping in `internal/workflow/workflow.go` (`inputMapping`, `aggregatedOutput`); handler
  `internal/handlers/workflow_schema.go`.
- Related: [ADR-0014](0014-schema-versioning-and-rollback.md),
  [ADR-0015](0015-conditional-routing-with-expr-lang.md),
  [ADR-0012](0012-join-merge-strategies.md), [ADR-0004](0004-multi-trigger-workflow-initiation.md)
  (`TriggerConfig` on the schema).
