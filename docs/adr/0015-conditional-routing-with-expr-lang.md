# 0015. Conditional edge routing with expr-lang

- Status: Accepted
- Date: 2026-06-01
- Deciders: FUSE maintainers

> Backfill ADR: records a foundational decision already implemented.

## Context and Problem Statement

Workflows need to branch: route to different downstream nodes based on a prior node's output
(if/switch logic), and trigger workflows only for events matching some predicate. We need an
expression mechanism that is expressive enough for real conditions yet safe to evaluate on
user-supplied schemas (no arbitrary code execution).

## Decision Drivers

- Express non-trivial conditions (comparisons, boolean logic) over node output / event data.
- Safe evaluation — no arbitrary code, sandboxed.
- Lightweight (pure Go, no embedded VM/JS runtime).
- A clear "otherwise" path so routing is total.

## Considered Options

- **expr-lang/expr expressions on edges, plus exact-match and default condition types.**
- **Embed a scripting engine** (JS/Lua) for conditions.
- **Exact value matching only** (no general expressions).

## Decision Outcome

Chosen option: **`expr-lang/expr` for expressions, alongside exact and default conditions.** An
`EdgeCondition` (`internal/workflow/edge_schema.go`) has a `Type`:

- `expression` — an `expr-lang` expression evaluated against an environment of all node outputs
  (keyed by node id) plus `output` for the current node (e.g. `output.status == "success" && total > 100`).
- `exact` — compare `Value` against the node's configured conditional output field.
- `default` — always matches; the fallback when no other edge matches.

`EvaluateCondition` (`internal/workflow/expression.go`) evaluates per type;
`filterOutputEdgesByConditionals` (`internal/workflow/workflow.go`) picks the matching output
edge(s): unconditional edges always pass, expression/exact edges are evaluated, and if none match
the `default` edge is used. The same expr-lang engine powers **event-trigger filters**
(`EventConfig.Filter`, `internal/actors/event_trigger.go`) — events are evaluated against their
data and only matching ones trigger the workflow ([ADR-0004](0004-multi-trigger-workflow-initiation.md)).

### Consequences

- Good: expressive, safe (sandboxed, no arbitrary code), pure-Go, fast to compile/evaluate.
- Good: one expression language for both edge routing and event filtering — one thing to learn.
- Good: `default` guarantees total routing (no "stuck" node when nothing matches).
- Bad: expression errors surface at runtime (a bad expression logs and skips the edge); authors
  must know expr-lang syntax.
- Neutral: edge-condition env and event-filter env are scoped differently (node outputs vs raw
  event data) — intentional.

## Pros and Cons of the Options

### expr-lang + exact + default (chosen)

- Good: expressive yet safe; pure Go; covers simple and complex branching; total routing.
- Bad: runtime evaluation errors; another mini-language for authors.

### Embedded scripting engine (JS/Lua)

- Good: maximal expressiveness/familiarity.
- Bad: heavy dependency; sandboxing/security burden; slower; overkill for routing predicates.

### Exact match only

- Good: trivial and safe.
- Bad: can't express comparisons/boolean logic — forces extra nodes for real conditions.

## More Information

- Code: `internal/workflow/edge_schema.go` (`EdgeCondition`, `EdgeConditionType`),
  `expression.go` (`EvaluateCondition`), `workflow.go` (`filterOutputEdgesByConditionals`),
  `internal/actors/event_trigger.go` (event filters); dependency `github.com/expr-lang/expr`.
- Related: [ADR-0013](0013-workflow-schema-model-and-input-mapping.md),
  [ADR-0004](0004-multi-trigger-workflow-initiation.md).
