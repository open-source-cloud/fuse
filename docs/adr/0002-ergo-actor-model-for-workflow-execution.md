# 0002. Ergo actor model for workflow execution

- Status: Accepted
- Date: 2026-06-01
- Deciders: FUSE maintainers

> Backfill ADR: records a foundational decision made early in the project.

## Context and Problem Statement

FUSE executes workflows as graphs of nodes, where each node runs a function and
passes data to the next. Execution must be concurrent (parallel branches), fault
tolerant (a failing node must not take down the engine), resumable (workflows can
sleep, wait on async results, and recover after a crash), and horizontally
scalable for HA. We needed a concurrency and supervision model to build this on.

## Decision Drivers

- Isolation between in-flight workflows and between node executions.
- Supervision / fault tolerance — restart failed units without crashing the process.
- Natural fit for message passing (trigger → execute → result → next).
- Support for long-running / async steps and crash recovery (journal replay).
- A path to clustering / HA across nodes.

## Considered Options

- **ergo.services actor model** — Erlang/OTP-style actors, supervisors, and
  distributed clustering for Go.
- **Hand-rolled goroutines + channels** — bespoke concurrency per subsystem.
- **External queue + stateless workers** (e.g. a broker with worker pods).

## Decision Outcome

Chosen option: **ergo.services actor model**. Each workflow instance and each
function execution is an actor under a supervision tree, communicating by
messages. This gives isolation, supervised restarts, and a built-in clustering
story, and it maps cleanly onto the trigger→execute→result→advance flow. Workflow
progress is persisted to a journal so an actor can replay and resume after
failure.

### Consequences

- Good: supervised fault tolerance and restart semantics out of the box; clean
  message-driven execution; clustering primitives available for HA.
- Good: long-running/async steps fit the mailbox model (a step can defer and be
  completed later by a message).
- Bad: contributors must learn the actor/OTP mental model and ergo's API.
- Bad: couples the engine to ergo.services; messages must use ergo-supported types.
- Neutral: actor state must be persisted (journal + repositories) to survive restarts.

## Pros and Cons of the Options

### ergo.services actors

- Good: supervision, isolation, message passing, and clustering are first-class.
- Bad: framework coupling; learning curve; some API sharp edges (e.g. `Send` typing).

### Goroutines + channels

- Good: no framework dependency; full control.
- Bad: we would reimplement supervision, restart, addressing, and clustering ourselves.

### External queue + stateless workers

- Good: scales horizontally; mature infra.
- Bad: heavy external dependency; per-node messaging and ordered intra-workflow state
  become awkward; more moving parts for local dev.

## More Information

- Implementation: `internal/actors/` — `WorkflowSupervisor`, `WorkflowInstanceSupervisor`,
  `WorkflowHandler`, `WorkflowFunc` (worker pool), plus trigger actors.
- Actor patterns are also documented in `.agents/rules/03-actor-patterns.mdc` (see [ADR-0009](0009-portable-ai-agent-guidance.md)).
- Related: [ADR-0003](0003-in-memory-repositories-by-default.md) (persistence for resume/HA),
  [ADR-0004](0004-multi-trigger-workflow-initiation.md) (how actors are triggered).
