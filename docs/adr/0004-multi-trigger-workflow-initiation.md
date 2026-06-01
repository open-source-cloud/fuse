# 0004. Multi-trigger workflow initiation

- Status: Accepted
- Date: 2026-06-01
- Deciders: FUSE maintainers

> Backfill ADR: records a foundational decision made early in the project.

## Context and Problem Statement

A workflow automation engine must start workflows from many kinds of stimuli:
direct API calls, schedules, inbound webhooks from third parties, and internal
events. We needed a uniform way to declare *how* a workflow is triggered and a
single execution path once it fires, including safe behavior under HA (no
duplicate runs when multiple nodes observe the same stimulus).

## Decision Drivers

- Support diverse initiation sources without bespoke wiring per workflow.
- One convergent path into the executor regardless of trigger type.
- Idempotency / deduplication, especially across HA nodes.
- Declared in the workflow schema, so triggering is part of the definition.

## Considered Options

- **A typed `TriggerConfig` on the schema** with `http`, `cron`, `webhook`, and
  `event` types, each handled by a dedicated handler/actor that converges on
  `WorkflowSupervisor`.
- **HTTP-only** — external schedulers/webhook relays call the trigger endpoint.
- **Per-trigger bespoke entry points** with no shared abstraction.

## Decision Outcome

Chosen option: **a typed `TriggerConfig` supporting HTTP, cron, webhook, and
event**. HTTP triggers via `POST /v1/workflows/trigger`; cron via a scheduler
actor; webhooks via a path-routing handler (with optional HMAC verification);
events via an internal event-bus subscriber (with optional expr-lang filters).
All converge on a single message to `WorkflowSupervisor`, which spawns the
workflow. Idempotency keys (explicit for HTTP, deterministic for cron/event)
provide deduplication, including across HA nodes. A UI form is just an HTTP or
webhook trigger.

### Consequences

- Good: new initiation sources slot in behind one abstraction; one execution path.
- Good: HA-safe via idempotency/claim-based dedup.
- Good: triggering is declarative, versioned with the schema.
- Bad: four trigger mechanisms to maintain and test.
- Neutral: webhook/event matching scans schemas to resolve the target workflow.

## Pros and Cons of the Options

### Typed TriggerConfig (http/cron/webhook/event)

- Good: covers the common automation sources natively; uniform downstream path.
- Bad: more surface area than HTTP-only.

### HTTP-only

- Good: minimal engine surface.
- Bad: pushes scheduling, webhook receipt, and event fan-in onto external systems.

### Bespoke per-trigger entry points

- Good: maximal flexibility per case.
- Bad: duplicated logic; no consistent idempotency/observability story.

## More Information

- Schema: `internal/workflow/trigger.go` (`TriggerConfig`, `CronConfig`,
  `WebhookConfig`, `EventConfig`).
- Handlers/actors: `internal/handlers/trigger_workflow.go`, `internal/handlers/webhook.go`,
  `internal/actors/cron_scheduler.go`, `internal/actors/event_trigger.go`.
- Related: [ADR-0002](0002-ergo-actor-model-for-workflow-execution.md) (the actor
  pipeline triggers feed), [ADR-0005](0005-ai-agents-as-workflow-nodes-phased-roadmap.md)
  (agents reuse these triggers unchanged).
