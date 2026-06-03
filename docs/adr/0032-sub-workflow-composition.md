# 0032. Sub-workflow composition: child workflows as first-class instances linked by a durable ref

- Status: Accepted
- Date: 2026-06-03
- Deciders: FUSE maintainers

## Context and Problem Statement

A workflow often needs to invoke **another workflow** as a step — to reuse a published pipeline,
decompose a large graph into modular pieces, or give a sub-pipeline its own schema version,
concurrency limit, and timeout. This needs a composition primitive that fits the actor model
([ADR-0002](0002-ergo-actor-model-for-workflow-execution.md)) and the durable-execution model
([ADR-0010](0010-durable-execution-journal-and-replay.md)): the parent must be able to launch a
child, optionally wait for it, survive a restart mid-flight, and work in a multi-node cluster
([ADR-0018](0018-high-availability-and-clustering.md)) where parent and child may live on different
nodes. This decision records the model already implemented; it was previously only mentioned in
passing by ADR-0010.

## Decision Drivers

- **Reuse & modularity** — invoke an existing workflow schema by id from within another workflow.
- **Isolation** — the child is its own execution with its own journal, threads, versioning,
  concurrency, and timeout, not code nested inside the parent's actor.
- **Durability & replay** — the parent↔child link and the start/complete events must survive
  restart and replay deterministically.
- **HA-friendly** — the child is triggered like any workflow, so it can run on any node; the link
  must be discoverable across nodes.
- **Reuse existing machinery** — lean on the trigger path, journal, and supervision tree rather
  than building a second nested-execution engine.

## Considered Options

- **A — Nested in-process execution.** Run the child graph inside the parent's `WorkflowHandler`.
- **B — Child as a first-class workflow instance linked by a persisted `SubWorkflowRef`** (chosen).
- **C — External orchestration only** (no in-engine composition; callers chain via the API).

## Decision Outcome

Chosen: **B.** A system function (`fuse/pkg/system/subworkflow`, `system.SubWorkflowFullFunctionID`)
emits a `RunSubWorkflowAction`; the parent's `WorkflowHandler.handleSubWorkflowAction`
(`internal/actors/workflow_handler.go`):

1. Mints a fresh child `workflow.ID` and persists a **`SubWorkflowRef`**
   (`internal/workflow/subworkflow.go`: `ParentWorkflowID`, `ParentThreadID`, `ParentExecID`,
   `ChildWorkflowID`, `ChildSchemaID`, `Async`) via `WorkflowRepository.SaveSubWorkflowRef`
   (Postgres table `sub_workflow_refs`, migration `000001`).
2. Appends a `subworkflow:started` journal entry (`JournalSubWorkflowStarted`) for replay.
3. Triggers the child through the **normal trigger path**
   (`messaging.NewTriggerWorkflowWithEnvMessage`), so the child is an ordinary workflow instance
   with its own actor tree, journal, version, concurrency, and timeout — and **inherits the
   parent's environment** ([ADR-0031](0031-settings-secrets-and-environments.md)).

On completion, the child's handler calls `notifyParentIfSubWorkflow`: it looks up its
`SubWorkflowRef` and sends a `SubWorkflowCompletedMessage` to the parent handler, which resumes
the waiting parent thread (`handleMsgSubWorkflowCompleted`) and records `subworkflow:completed`.
The **`Async` flag** distinguishes a **synchronous** sub-workflow (the parent thread blocks until
the child completes, then continues with the child's output) from an **asynchronous** one
(fire-and-forget; the parent does not wait). Because the link is persisted, recovery can find
in-flight children (`FindActiveSubWorkflows`) and the completion notification works across nodes.

### Consequences

- Good: full reuse and isolation — children are independently versioned, rate-limited, timed out,
  observed, and replayed; the parent graph stays small.
- Good: durable and HA-safe — the persisted `SubWorkflowRef` + journal entries survive restart and
  let parent/child live on different nodes.
- Good: minimal new machinery — composition rides the existing trigger, journal, and supervision
  paths.
- Bad: extra coordination surface — completion notification by message, ref persistence, and the
  possibility of orphaned children if a parent is cancelled mid-flight (mitigated by recovery
  sweeps).
- Neutral: cross-node completion relies on actor messaging by name; a lost message must be healed
  by recovery rather than a synchronous call.

## Pros and Cons of the Options

### A — Nested in-process execution
- Good: no cross-instance coordination; simplest happy path.
- Bad: no isolation (shared journal/threads), no independent versioning/concurrency/timeout, and
  replay of a nested graph inside the parent is far more complex. Rejected.

### B — First-class instance + durable ref (chosen)
- Good: isolation, reuse, durability, HA; reuses trigger/journal/supervision.
- Bad: coordination + persistence overhead; orphan handling.

### C — External orchestration only
- Good: zero engine complexity.
- Bad: pushes composition to every caller; no in-engine durability/replay for the parent↔child
  relationship. Rejected as the in-engine primitive.

## More Information

- Code: `internal/workflow/subworkflow.go` (`SubWorkflowRef`), `internal/actors/workflow_handler.go`
  (`handleSubWorkflowAction`, `notifyParentIfSubWorkflow`, `handleMsgSubWorkflowCompleted`,
  `handleSystemSubWorkflow`), `internal/workflow/journal.go` (`subworkflow:started|completed`),
  `internal/workflow/workflowactions` (`RunSubWorkflowAction`), Postgres `sub_workflow_refs`
  (`FindSubWorkflowRef`, `FindActiveSubWorkflows`, `SaveSubWorkflowRef`).
- Related: [ADR-0010](0010-durable-execution-journal-and-replay.md) (journal/replay the start/complete
  events ride on), [ADR-0011](0011-threading-model-and-foreach.md) (the parent thread that blocks on
  a sync child), [ADR-0018](0018-high-availability-and-clustering.md) (cross-node parent/child),
  [ADR-0023](0023-timeout-enforcement-model.md) (each instance times out independently),
  [ADR-0031](0031-settings-secrets-and-environments.md) (children inherit the parent's environment),
  and [ADR-0027](0027-async-tool-invocation-sub-execution-channel.md) (a distinct, agent-driven
  async sub-execution channel — not the same as workflow-graph sub-workflows).
