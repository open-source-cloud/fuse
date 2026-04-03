# Fuse Workflow Engine — Roadmap & Specifications

## Project Vision

Fuse is a workflow engine built on the **ergo actor model** for creating automation pipelines. Each workflow node runs as an independent actor with message passing, supervision, and fault tolerance.

The long-term vision (from [#1](https://github.com/open-source-cloud/fuse/issues/1)) is a workflow management system that:

- Creates workflows using a graph structure (UI-extensible via API)
- Executes workflows end-to-end, both synchronously and asynchronously
- Respects ACID properties (Atomicity, Consistency, Isolation, Durability)
- Supports asynchronous, atemporal, and event-based FSM patterns
- Enables multi-level composition (nodes can be other workflows)

---

## Current State (What Exists Today)

### Implemented Features

| Feature | Issue | Status |
|---------|-------|--------|
| Conditional branching | [#10](https://github.com/open-source-cloud/fuse/issues/10) | Done |
| Parallel branching | [#11](https://github.com/open-source-cloud/fuse/issues/11) | Done |
| Async node execution | [#12](https://github.com/open-source-cloud/fuse/issues/12), [#33](https://github.com/open-source-cloud/fuse/issues/33) | Done |
| Start workflow endpoint | [#14](https://github.com/open-source-cloud/fuse/issues/14) | Done |
| Workflow schema persistence | [#16](https://github.com/open-source-cloud/fuse/issues/16) | Done |
| Server actor + app supervisor | [#20](https://github.com/open-source-cloud/fuse/issues/20) | Done |
| Workflow loops | [#22](https://github.com/open-source-cloud/fuse/issues/22) | Done |
| HTTP node provider | [#24](https://github.com/open-source-cloud/fuse/issues/24) | Done |
| External node providers | [#25](https://github.com/open-source-cloud/fuse/issues/25) | Done |
| Entity renaming + schema refactors | [#26](https://github.com/open-source-cloud/fuse/issues/26), [#28](https://github.com/open-source-cloud/fuse/issues/28) | Done |
| Store + PrivateStore | [#29](https://github.com/open-source-cloud/fuse/issues/29) | Done |
| API fixes + WorkflowID return | [#38](https://github.com/open-source-cloud/fuse/issues/38) | Done |
| External packages + Registry API | [#41](https://github.com/open-source-cloud/fuse/issues/41) | Done |

### Open Issues

| Issue | Title | Status |
|-------|-------|--------|
| [#1](https://github.com/open-source-cloud/fuse/issues/1) | Define project structure and vision | Open |
| [#34](https://github.com/open-source-cloud/fuse/issues/34) | Audit tracing (for devmode) | Open |

### Current Architecture

```
                                    HTTP Request
                                         |
                                    MuxServerSup
                                    /          \
                              MuxServer    WorkerPools (HTTP handlers)
                                               |
                                    TriggerWorkflowHandler
                                               |
                                    WorkflowSupervisor
                                         |  (spawns per workflow)
                                    WorkflowInstanceSup
                                    /                  \
                           WorkflowHandler        WorkflowFuncPool
                           (orchestrator)         /       |       \
                                            WkFunc   WkFunc    WkFunc
                                           (workers execute functions)
```

**Workflow execution model:**
```
Trigger() -> RunFunctionAction -> WorkflowFunc executes -> FunctionResult
     -> SetResultFor() -> Next(threadID) -> RunFunctionAction -> ...
     -> (all threads finished) -> StateFinished
```

**Control flow patterns supported:**
- Sequential: Single edge chains
- Conditional branching: Edge conditions with value matching
- Parallel branching: Multiple edges fork into parallel threads
- Join/convergence: Multiple input edges synchronize via `AreAllParentsFinishedFor()`
- Loops: Graph cycles (with thread-safe traversal)
- Async functions: External completion via HTTP callback

### Core Types (Current)

```go
// Workflow states
StateUntriggered | StateRunning | StateSleeping | StateFinished | StateError

// Action types (what Next() returns)
ActionNoop | ActionRunFunction | ActionRunParallelFunctions

// Message types (actor communication)
TriggerWorkflow | ExecuteFunction | FunctionResult | AsyncFunctionResult

// Thread states
ThreadRunning | ThreadFinished
```

---

## Concepts & Terminology Glossary

These concepts are introduced across the roadmap phases. They draw from patterns in **n8n**, **Inngest**, and **Restate**.

| Concept | Description | Inspiration |
|---------|-------------|-------------|
| **Durable Execution** | Workflow state is persisted at each step boundary. On crash/restart, execution replays from the last checkpoint instead of re-running completed steps. | Restate, Inngest |
| **Execution Journal** | An append-only log of step results that enables deterministic replay. Each completed step writes its result to the journal; on replay, results are read from the journal instead of re-executing. | Restate |
| **Step Checkpoint** | A boundary in execution where state is durably persisted. If execution fails after a checkpoint, it resumes from the checkpoint rather than starting over. | Inngest |
| **Awakeable / Wait-for-Event** | A durable promise that pauses workflow execution until an external system resolves it. Similar to Fuse's current async function pattern, but with timeout and cancellation support. | Restate |
| **Durable Timer** | A sleep/delay that survives process restarts. The timer deadline is persisted; on restart, the engine schedules wake-up at the correct future time. | Restate, Inngest |
| **Error Edge** | A special edge type that activates when the source node fails, routing to recovery/compensation logic instead of terminating the workflow. | n8n |
| **Retry Policy** | Per-node configuration defining how many times and with what backoff strategy a failed function should be retried before triggering error handling. | Inngest, Restate |
| **Sub-workflow** | A workflow that runs as a node within a parent workflow. The parent sleeps until the child completes, then receives the child's output. | n8n |
| **Merge Strategy** | Configuration at join/convergence nodes defining how outputs from multiple parent branches are combined (append, merge, pick-first, etc.). | n8n |
| **Concurrency Control** | Limits on how many instances of a function can execute in parallel, preventing overload of external systems. | Inngest |
| **Idempotency Key** | A client-provided key that ensures triggering a workflow with the same key returns the existing execution instead of creating a duplicate. | Inngest, Restate |
| **Workflow Versioning** | Schema versions that allow running workflows to use the schema version they started with, while new workflows use the latest version. | n8n |

---

## Phase Overview

The roadmap is organized into 4 phases, each building on the previous:

```
Phase 1: Foundation (Make It Reliable)
   |
   |-- Durable Execution / Workflow Resumption
   |-- Retry & Error Recovery
   |-- Timeouts
   |-- Graceful Workflow Completion
   |-- Input Mapping Validation
   |
Phase 2: Control Flow (Make It Powerful)
   |  depends on: Phase 1 (durability for sleep/wait, retries for sub-workflows)
   |
   |-- Wait / Sleep / Delayed Execution
   |-- Workflow Cancellation
   |-- Sub-workflows
   |-- Enhanced Conditionals / Switch
   |-- Merge Strategies at Join Nodes
   |
Phase 3: Operational (Make It Production-Ready)
   |  depends on: Phase 1 (durability), Phase 2 (cancellation)
   |
   |-- Event-Driven Triggers
   |-- Concurrency Control
   |-- Throttling & Rate Limiting
   |-- Idempotency
   |-- Audit Tracing Persistence
   |
Phase 4: Polish (Make It Great)
   |  depends on: Phase 1-3
   |
   |-- Loop Over Items / Batch Processing
   |-- Workflow Versioning
```

### Dependency Graph Between Features

```
                    Durable Execution (1.1)
                   /        |            \
           Retry (1.2)  Timeouts (1.3)  Graceful Completion (1.4)
              |             |
        Sleep/Wait (2.1)    |
              |             |
        Sub-workflows (2.3) |
              |             |
        Cancellation (2.2)--+
              |
    Event Triggers (3.1)
              |
    Concurrency (3.2) --> Throttling (3.3)
              |
    Idempotency (3.4)
              |
    Audit Persistence (3.5)
              |
    Batch Processing (4.1)
              |
    Versioning (4.2)
```

---

## Phase Documents

| Phase | Document | Description |
|-------|----------|-------------|
| 1 | [phase-1-foundation.md](./phase-1-foundation.md) | Durable execution, retries, timeouts, completion, validation |
| 2 | [phase-2-control-flow.md](./phase-2-control-flow.md) | Sleep/wait, cancellation, sub-workflows, switch, merge |
| 3 | [phase-3-operational.md](./phase-3-operational.md) | Triggers, concurrency, throttling, idempotency, audit |
| 4 | [phase-4-polish.md](./phase-4-polish.md) | Batch processing, versioning |

---

## Reference Projects

Throughout these specs, we reference three open-source projects as prior art:

- **[n8n](https://github.com/n8n-io/n8n)** — Visual workflow automation platform with 400+ integrations, sub-workflows, error handling flows, and a merge node for parallel branch consolidation.
- **[Inngest](https://github.com/inngest/inngest)** — Event-driven durable execution engine with step-level checkpointing, automatic retries, flow control primitives (sleep, wait-for-event, throttling, rate limiting, concurrency limits).
- **[Restate](https://github.com/restatedev/restate)** — Durable execution runtime with execution journaling, deterministic replay, virtual objects with K/V state, awakeables for external signals, and durable timers.
