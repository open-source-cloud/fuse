# 0012. Configurable merge strategies at join nodes

- Status: Accepted
- Date: 2026-06-01
- Deciders: FUSE maintainers

> Backfill ADR: records a foundational decision already implemented.

## Context and Problem Statement

When parallel branches ([ADR-0011](0011-threading-model-and-foreach.md)) reconverge at a join
node, their outputs must be combined into a single input for the downstream node. Different
workflows need different semantics — collect into a list, merge into one object, take one
branch's result, or key results by branch. A single hard-coded merge behavior would force
awkward workarounds.

## Decision Drivers

- Cover the common aggregation shapes without custom code per workflow.
- Make the behavior explicit and declarative on the node.
- Sensible default when unspecified.

## Considered Options

- **A fixed set of named merge strategies** selected via node config.
- **Always merge into one object** (single behavior).
- **A user expression** to compute the merged value.

## Decision Outcome

Chosen option: **a fixed set of named strategies** configured per join node via `MergeConfig`
(`internal/workflow/merge.go`), validated to one of: `append`, `merge`, `first`, `last`, `keyed`.
`ApplyMergeStrategy` combines the branch outputs (each a `BranchInput` of `EdgeID`, `ThreadID`,
`Data`):

- **append** (default) — concatenate compatible slice values; otherwise collect per-key values
  into arrays (`{count:1}+{count:2}` → `{count:[1,2]}`).
- **merge** — shallow object merge; later branch overrides earlier on key conflicts.
- **first** — keep only the first branch's output.
- **last** — keep only the last branch's output.
- **keyed** — group outputs by edge id into a nested map (`{"branch-a":{…},"branch-b":{…}}`).

`DefaultMergeConfig` returns `append`.

### Consequences

- Good: the common fan-in shapes are one config field; no per-workflow merge code.
- Good: explicit and validated; predictable defaults.
- Bad: not arbitrarily expressive — a need outside the five requires a new strategy or an
  extra transform node.
- Neutral: `append`'s slice-vs-array coercion rules are subtle; documented in code.

## Pros and Cons of the Options

### Fixed named strategies (chosen)

- Good: declarative, validated, covers the common cases, good default.
- Bad: bounded expressiveness.

### Single always-merge-object behavior

- Good: simplest.
- Bad: can't express list collection or branch selection; forces workarounds.

### User expression per join

- Good: maximal flexibility.
- Bad: more complex/error-prone; overkill for the common shapes; harder to validate.

## More Information

- Code: `internal/workflow/merge.go` (`MergeConfig`, `MergeStrategyType`, `BranchInput`,
  `ApplyMergeStrategy`, `DefaultMergeConfig`); configured via `NodeSchema.Merge`
  ([ADR-0013](0013-workflow-schema-model-and-input-mapping.md)).
- Related: [ADR-0011](0011-threading-model-and-foreach.md) (produces the parallel branch outputs).
