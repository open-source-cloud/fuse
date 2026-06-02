# 0020. Observability: Prometheus metrics, OpenTelemetry tracing, execution traces

- Status: Accepted
- Date: 2026-06-01
- Deciders: FUSE maintainers

> Backfill ADR: records a foundational decision already implemented.

## Context and Problem Statement

Operators need to monitor a fleet of FUSE nodes (throughput, failures, latency) and debug
individual workflow runs (what ran, with what input/output, how long, why it failed). These are
two different audiences — aggregate metrics and per-execution detail — and tracing may or may not
have a collector available. We need an approach that serves both and degrades gracefully.

## Decision Drivers

- Standard, scrapeable metrics for dashboards/alerts.
- Distributed tracing that's optional (works with no collector configured).
- Per-execution, queryable debugging data without requiring an external tracing stack.
- Isolation from global registries / no nil-checks scattered in code.

## Considered Options

- **Three complementary layers: Prometheus metrics + optional OTel tracing + a persisted
  execution trace.**
- **Metrics only.**
- **OpenTelemetry for everything** (metrics + traces) via a required collector.

## Decision Outcome

Chosen option: **three layers, each for its job.**

1. **Metrics (Prometheus).** A dedicated `FuseMetrics` registry (`internal/metrics/registry.go`)
   exposes `fuse_*` series — `workflows_active` (gauge), `workflows_completed/failed/cancelled_total`
   (counters), `node_exec_duration_seconds` (histogram, labels `function_id`, `status`) — plus an
   ergo runtime collector (`ergo_*`) and Go/process collectors, served at `GET /metrics`
   (OpenMetrics) by the mux server. A dedicated registry avoids polluting global Prometheus state.

2. **Tracing (OpenTelemetry).** `internal/tracing/provider.go` exports spans over OTLP/gRPC when
   `OTEL_ENABLED=true`; when disabled it returns a **no-op provider** so call sites never nil-check.
   W3C TraceContext + Baggage propagators let span context ride on actor messages
   (`InjectCarrier`/`ExtractCarrier`): a root span per workflow, child `node.execute` spans with
   workflow/exec/function/package attributes.

3. **Execution trace (domain).** `ExecutionTrace`/`ExecutionStepTrace` (`internal/workflow/trace.go`)
   is a persisted, per-run record (status, timings, per-step input/output/attempt/error), stored
   via Postgres + object store ([ADR-0019](0019-object-store-payload-externalization.md)) and
   queryable at `GET /v1/workflows/{id}/trace` — debugging detail with **no external collector**.

### Consequences

- Good: aggregate monitoring (Prometheus), distributed tracing when wanted (OTel), and always-on
  per-run debugging (execution trace) — each audience served.
- Good: tracing is optional (no-op fallback); metrics registry is isolated.
- Bad: the execution trace overlaps conceptually with OTel spans (two "trace" notions) — kept
  because it's queryable without a collector and persisted with the workflow.
- Neutral: OTel sampling/retention for execution traces aren't yet centrally tuned.

## Pros and Cons of the Options

### Three layers (chosen)

- Good: right tool per need; graceful degradation; no hard dependency on a tracing stack.
- Bad: two "trace" concepts to understand; more surface area.

### Metrics only

- Good: simplest.
- Bad: no request-level debugging or distributed correlation.

### OTel for everything (required collector)

- Good: one standard, unified.
- Bad: forces a collector to exist; loses the always-available, persisted-with-the-workflow
  execution trace.

## More Information

- Code: `internal/metrics/registry.go`, `ergo_collector.go`; `internal/actors/mux_server.go`
  (`/metrics`); `internal/tracing/provider.go`; span use in
  `internal/actors/workflow_handler.go` + `workflow_func.go`; `internal/workflow/trace.go`
  (+ `postgres/trace.go`). Config: `OtelConfig` (`internal/app/config/config.go`).
- Related: [ADR-0019](0019-object-store-payload-externalization.md),
  [ADR-0010](0010-durable-execution-journal-and-replay.md) (journal vs trace: replay vs observability).
