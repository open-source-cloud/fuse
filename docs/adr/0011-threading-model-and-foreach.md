# 0011. Thread model for fork/join and ForEach iteration

- Status: Accepted
- Date: 2026-06-01
- Deciders: FUSE maintainers

> Backfill ADR: records a foundational decision already implemented.

## Context and Problem Statement

A workflow graph can fork into parallel branches, join them back, and loop over collections
(ForEach). The engine needs a way to track concurrent execution paths, route async results back
to the correct path, and know when parallel branches have all completed so a join can proceed —
all while remaining replayable from the journal ([ADR-0010](0010-durable-execution-journal-and-replay.md)).

## Decision Drivers

- Represent concurrent execution paths within a single workflow instance.
- Route a late/async result to the exact path that produced it.
- Deterministic join semantics (a join waits for its real parents).
- Support dynamic, runtime-sized fan-out (ForEach over N items).

## Considered Options

- **Per-path "threads" with IDs encoded into the execution ID**, static IDs assigned at
  graph-compile time and dynamic IDs allocated at runtime for ForEach.
- **Goroutine-per-branch with channels** for join coordination.
- **A separate child workflow per branch/iteration** (reuse sub-workflows for all fan-out).

## Decision Outcome

Chosen option: **explicit per-path threads**. Each execution path is a `thread`
(`internal/workflow/thread.go`) with a `uint16` id and state (running/finished). Thread ids are
assigned by a two-phase algorithm during graph compilation (`internal/workflow/graph.go`): a
DFS assigns ids (trigger = thread 0, a new thread per fork branch and guessed parents at joins),
then a stabilization pass fixes each join's `parentThreads` to the real reachable parents
(dropping "ghost" threads from cycles). A join proceeds only when all its `parentThreads` are
finished.

The **thread id is embedded in the `ExecID`** (UUIDv8, bytes 6–7, `pkg/workflow/exec_id.go`),
so any async result — delivered later via `Finish` or an awakeable — carries the thread it
belongs to and routes back to the correct path.

**ForEach** (`internal/workflow/foreach.go` + `foreach_state.go`) builds on this with **dynamic
thread allocation**: each iteration/batch gets a fresh thread via `threads.AllocateDynamicID()`,
runs the loop body down the node's `each` edge, and records its result in `ForEachState`. Up to
a configured concurrency run in parallel; when all complete, a `each → done` phase transition
emits the aggregated `results` down the `done` edge. Merge of parallel branch outputs at a join
uses the strategies in [ADR-0012](0012-join-merge-strategies.md).

### Consequences

- Good: concurrency is explicit, replayable, and addressable; async results route precisely.
- Good: ForEach fan-out is runtime-sized without predeclaring iteration count.
- Bad: thread ids are bounded by the 12-bit ExecID field (max 4095 concurrent threads per
  workflow) — ample in practice but a real ceiling.
- Neutral: the two-phase thread algorithm is non-obvious (documented in code + this ADR).

## Pros and Cons of the Options

### Per-path threads with ids in ExecID (chosen)

- Good: precise async routing, deterministic joins, replayable, supports dynamic fan-out.
- Bad: bespoke thread bookkeeping; 4095-thread ceiling per workflow.

### Goroutine + channels per branch

- Good: idiomatic Go concurrency.
- Bad: not replayable from a journal; join/await state lives in memory and dies on restart.

### Child workflow per branch/iteration

- Good: reuses one mechanism; strong isolation.
- Bad: heavy for fine-grained fan-out (a workflow instance per item); more scheduling overhead.

## More Information

- Code: `internal/workflow/thread.go`, `graph.go` (thread calculation), `foreach.go`,
  `foreach_state.go`; `pkg/workflow/exec_id.go` (thread id encoding).
- Related: [ADR-0010](0010-durable-execution-journal-and-replay.md) (journal records
  thread:created/finished and foreach:* entries), [ADR-0012](0012-join-merge-strategies.md).
