---
name: observability-tracing
description: Explains FUSE's three observability layers — OTel spans, Prometheus metrics, and the persisted execution trace — with the exact span names, attribute conventions and carrier helpers in use, the three trace-context breaks A-04 owns with file:line, and how to verify a trace end to end against a real collector. Use when touching internal/tracing, internal/metrics, internal/logging, span or metric call sites, or when working A-04 or C-04.
---

# observability-tracing

Backlog ids below are from the repo-root backlog (`BACKLOG_V2.md`). *If it has been renamed to
`BACKLOG.md`, the canonical file is the one titled "FUSE — product shape and backlog".*

## When to use

- Adding or moving a span, an attribute, a metric, or a log call.
- A-04 (trace context end to end) and C-04 (GenAI telemetry and cost).
- Any change that crosses an actor boundary and should stay in one trace.

## Ground truth

| File | Authoritative for |
| --- | --- |
| `internal/tracing/provider.go` | The **entire** OTel surface — 115 lines, `NewProvider` + 4 exported methods |
| `internal/actors/workflow_handler.go` | The `workflow.execute` root span, every `InjectCarrier` call site, the workflow counters, `persistTrace` |
| `internal/actors/workflow_func.go` | The `node.execute` child span and the discarded context (L96) |
| `internal/messaging/message.go:39-45` | `Message.TraceCarrier` — the only trace-context transport in the system |
| `internal/metrics/registry.go` | The whole `fuse_*` metric set, in a dedicated registry |
| `internal/metrics/ergo_collector.go` | The `ergo_*` gauges over `gen.Node.Info()` |
| `internal/workflow/trace.go`, `trace_builder.go` | The persisted `ExecutionTrace` (the *other* notion of "trace") |
| `internal/logging/app_logger.go` | The global `logFormat` every other logger reads |
| `docs/adr/0020-observability-metrics-tracing-execution-traces.md` | Why three layers, and that the two "trace" notions overlap |
| `docs/adr/0029-llm-cost-and-usage-tracking-and-budgets.md` | Phase A (usage metrics) shipped; budgets deferred |

## ADR-0020's three layers, and the naming collision

1. **Prometheus metrics** — always on, scraped at `GET /metrics`.
2. **OTel spans** — optional (`OTEL_ENABLED`, default `false`), needs a collector.
3. **Persisted `ExecutionTrace`** — always available, queryable over REST, built from the journal.

(2) and (3) are both called "trace" and are unrelated. Nothing links them: migration
`000006_create_execution_traces` has **no `trace_id` or `span_id` column**, and
`internal/workflow/trace.go` has no such field. Given an `ExecutionTrace` there is no way to find the
corresponding OTel trace. If A-04 or C-04 wants that link it is a new column and a new migration.

## Provider setup

`tracing.NewProvider(cfg)` is provided by `CommonModule` (`internal/app/di/di.go`) and shut down from
`internal/app/fuse.go` (`(*Fuse).Terminate` → `tracingProvider.Shutdown`).

```go
// internal/tracing/provider.go — the complete exported surface
func (p *Provider) StartSpan(ctx, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span)  // L90
func (p *Provider) InjectCarrier(ctx) map[string]string                                                    // L95
func (p *Provider) ExtractCarrier(ctx, carrier map[string]string) context.Context                          // L102 — early-returns ctx when len==0
func (p *Provider) Shutdown(ctx) error                                                                     // L110 — nil-tp safe
```

Enabled path (L34-80): `otlptracegrpc` exporter, `semconv v1.30.0` ServiceName/ServiceVersion, a
composite `TraceContext{} + Baggage{}` propagator, and `otel.SetTracerProvider`/`SetTextMapPropagator`.
Config: `OTEL_ENABLED` (default `false`), `OTEL_EXPORTER_OTLP_ENDPOINT` (default `localhost:4317`),
`OTEL_SERVICE_NAME` (`fuse`), `OTEL_SERVICE_VERSION` (`unknown`), `OTEL_EXPORTER_OTLP_INSECURE` (`true`).

**Access tracing only through `*tracing.Provider`.** No code in the repo calls `otel.Tracer(...)` or
`otel.GetTextMapPropagator()` directly. It is a concrete struct, not an interface, so it cannot be
faked — unit tests must build a real provider from a config.

## Conventions in use (this is the complete list)

**Span names — exactly two:**

| Name | Created at | Ended at |
| --- | --- | --- |
| `workflow.execute` | `workflow_handler.go:213-221` `startRootSpan()`, called from L143 (existing-workflow / replay branch) and L189 (fresh create) | `sendWorkflowCompleted()` L511 |
| `node.execute` | `workflow_func.go:90-95` | L113, L131, L166 (three exit paths) |

**Attribute keys — exactly six**, dotted lowercase, always `attribute.String`:
`workflow.id`, `workflow.schema_id` (root span, L217-218), `workflow.final_state` (L510),
`node.exec_id`, `node.function_id`, `node.package_id` (`workflow_func.go:92-94`; the node span also
repeats `workflow.id` at L91). One event exists:
`nodeSpan.AddEvent("node.error", …)` at `workflow_func.go:160`.

Status/error idiom: `span.SetStatus(codes.Ok, "")` / `codes.Error` plus `span.RecordError(err)` —
see `workflow_func.go:111-113, 129-131, 157-166`.

**Trace context crosses actor boundaries as `map[string]string`, never as a `context.Context`** (ergo
messages are values):

```go
// sender   internal/actors/workflow_handler.go:699
messaging.NewExecuteFunctionMessage(id, execAction, env, a.tracingProvider.InjectCarrier(a.spanCtx))
// receiver internal/actors/workflow_func.go:89
parentCtx := a.tracingProvider.ExtractCarrier(context.Background(), msg.TraceCarrier)
```

`NewExecuteFunctionMessage` is the **only** message constructor that accepts a carrier. Adding one
elsewhere means adding the parameter to that constructor in `internal/messaging/`.

## The three A-04 breaks, with exact locations

### Break 1 — inbound extraction is missing at the HTTP trigger

`internal/handlers/trigger_workflow.go:HandlePost` (L65) has the `*http.Request` in hand and **never
reads `r.Header`** (grep confirms zero `r.Header` references in that file). It sends at L98 via
`messaging.NewTriggerWorkflowWithEnvMessage(schemaID, workflowID, environment)` —
`internal/messaging/trigger_workflow.go:30`, which takes no carrier, so `Message.TraceCarrier` is nil.
`workflow_sup.go` reads only `msg.TriggerWorkflowMessage()` (case at L98, read at L99) and calls
`spawnWorkflowActor(schemaID, workflowID, environment)` at L118 (declared at L239). The run's root span is then started
from `context.Background()` (`workflow_handler.go:214-216`) — a brand-new trace, disconnected from the
caller.

`ExtractCarrier` has exactly **one** production call site in the whole repo (`workflow_func.go:89`),
so it is genuinely never called at any HTTP boundary.

**Every path that starts a run has this break — there are seven, not four:**

| Entry point | Send site |
| --- | --- |
| HTTP trigger | `internal/handlers/trigger_workflow.go:98` |
| Webhook | `internal/handlers/webhook.go:97` |
| Retry from scratch | `internal/handlers/retry_workflow.go:94` |
| Event bus | `internal/actors/event_trigger.go:103` |
| Cron | `internal/actors/cron_scheduler.go:95` (no inbound parent — this is the one that needs a *documented* root) |
| HA claim sweep | `internal/actors/workflow_claim_actor.go:121` |
| Sub-workflow spawn | `internal/actors/workflow_handler.go:929` — already has a live parent in `a.spanCtx`, so it is the cheapest to parent correctly |

⚠️ Threading a carrier through the spawn chain touches
`internal/actors/workflow_instance_sup.go:56` (`if len(args) != 3`) and
`WorkflowHandlerInitArgs` (`workflow_handler.go:104`), which currently carries schemaID/workflowID/environment.

⚠️ `ExtractCarrier` accepts only `map[string]string`; there is **no** `http.Header` helper. Do not
convert `r.Header` naively — W3C keys are lowercase (`traceparent`, `tracestate`) while Go
canonicalises header names, and `propagation.MapCarrier` is a plain map lookup. Add a header-aware
helper on `Provider` (wrapping `propagation.HeaderCarrier`) rather than converting at each call site.

### Break 2 — handler → worker already works; it needs a regression test

Inject at `workflow_handler.go:699` (dispatch), `:663` (retry, via `SendAfter`) and `:1044`
(`spawnForEachBatch`) → `messaging.Message.TraceCarrier` → extract at `workflow_func.go:89`.
**Leave the mechanism alone.** There is no test of it today: `internal/actors` contains only
`event_trigger_test.go`, `event_trigger_dedup_test.go`, `execution_timer_test.go`, `mux_server_test.go`.

⚠️ **A regression test written against the default provider will pass while the code is broken.**
`noopProvider()` (`provider.go:82-87`) sets `prop: propagation.NewCompositeTextMapPropagator()` — a
composite with **zero** propagators — so with `OTEL_ENABLED=false` `InjectCarrier` returns an empty map
and `ExtractCarrier` returns the context unchanged. `internal/tracing/provider_test.go:49-61` admits
this in a comment ("A noop context produces an empty carrier"). A real test must build a provider with
`Otel.Enabled=true`, or otherwise install a real `propagation.TraceContext`.

### Break 3 — the node span context is discarded

`internal/actors/workflow_func.go:96` is exactly `_ = nodeCtx`. `nodeCtx` is bound at L90 by
`StartSpan(parentCtx, "node.execute", …)`; `nodeSpan` **is** used (L111-113, L129-131, L159-166) — only
the context is thrown away. It appears exactly once in the repo.

`pkg/workflow/execution_info.go:14-23` has no `context.Context` field, so node code cannot create a
child span or inject `traceparent`. Confirmed: the literal string `traceparent` appears **nowhere** in
the Go tree. `internal/packages/functions/http/request.go` builds headers solely from the node's
`headers` input, and `pkg/http/client.go` creates its own `context.Background()` with a timeout —
there is no caller-context seam at all. `go.mod` carries no `otelhttp`/contrib package, so inbound and
outbound propagation must be hand-rolled or a dependency added.

### Why A-04 is critical rather than nice-to-have

This is the seam **every SDK crosses**. B-02 (implement the remote transport) must inject
`traceparent` into the outbound invocation envelope specified by B-01, and it cannot: there is no
context to inject from. A distributed engine whose selling point is traceability, that breaks the
trace at exactly the boundary it invites you to extend across, is unsellable. A-04's third item —
"add the context to `ExecutionInfo`" — is therefore a **public `pkg/` API change**, which per the
backlog's rules needs a changelog entry and a migration path.

⚠️ **Tension to resolve in the A-04 ADR, not silently.**
`docs/adr/0027-async-tool-invocation-sub-execution-channel.md` (Status: Proposed) states explicitly
that per-execution runtime capability *must not* be added back as a field on
`workflow.ExecutionInfo` — a prior `Handle any` field was removed for exactly that reason. A-04 wants
a context field there. Address the conflict in the ADR.

## Metrics surface

`internal/metrics/registry.go` — a **dedicated** `prometheus.Registry`, never the global default.
A new series is a field on `FuseMetrics`, constructed in `NewFuseMetrics`, and appended to the
`reg.MustRegister(...)` list.

| Series | Type | Labels |
| --- | --- | --- |
| `fuse_workflows_active` | Gauge | — |
| `fuse_workflows_completed_total` / `_failed_total` / `_cancelled_total` | Counter | — |
| `fuse_node_exec_duration_seconds` | HistogramVec (DefBuckets) | `function_id`, `status` |
| `fuse_llm_tokens_total` | CounterVec | `function`, `provider`, `model`, `type` |
| `fuse_llm_calls_total` | CounterVec | `function`, `provider`, `model`, `status` |
| `ergo_*` (uptime, processes, memory, …) | Collector over `gen.Node.Info()` | `node` |

Scrape path: `mux_server.go` registers the ergo collector into the Fuse registry, then serves
`promhttp.HandlerFor(prometheus.Gatherers{fuseRegistry, combinedRegistry}, …)` at `GET /metrics`
(the combined registry adds the Go and process collectors). No handler is wrapped in tracing middleware.

**Packages that must not import prometheus declare a narrow port instead.**
`internal/packages/functions/ai/usage.go` defines `UsageRecorder` (`RecordUsage`, `RecordCall`) plus
`NopUsageRecorder`; `internal/packages/usage_recorder.go` is the metrics-backed adapter. Follow that
pattern for any C-04 recorder that must reach spans from the `ai` package.

## C-04 — what exists, what is missing, and the open question

**Exists (TODAY):** the two span names and six attributes above; LLM usage as Prometheus counters
only, via the `UsageRecorder` port, recorded from `ai/chat.go`, `ai/agent.go` and `ai/structured.go`.

**Missing (PROPOSED, C-04):** agent id, model, tokens in/out, cost, gate decision, tenant. Concrete
blockers per attribute:

| Wanted | Blocker |
| --- | --- |
| graph node id | `messaging.ExecuteFunctionMessage` has no node id field. It is recoverable handler-side via `a.workflow.AuditLog().Get(execID.String()).FunctionNodeID` — the pattern already used at `workflow_handler.go:693` to look up the execution timeout — so adding a field to the message is the mechanical route. |
| model, tokens, agent id | need a span reachable from `internal/packages/functions/ai`, which today has **no** tracing dependency at all. Depends on A-04 break 3. |
| cost | **no data source.** `pkg/llm` `Usage` carries prompt/completion/total tokens only; ADR-0029 defers the pricing table. |
| tenant | no tenancy concept exists in Core (it is Tier D, `D-01`). |

**The open question C-04 must answer first: which attributes belong to the engine, and which to the
caller/gateway.** The backlog's own guidance is "follow OTel GenAI conventions where they exist;
namespace the rest" and "meter cost at the gateway" — and the model gateway is **C-03**, which C-04
depends on. Cost per se cannot be computed in Core today. Decide the split in an ADR before writing
code; do not let C-04 silently absorb the adjacent undecided items (OTel sampling and
`TraceRetentionConfig`, declared at `internal/workflow/trace.go:37` and referenced nowhere else in
the Go tree — grep returns only the type and its doc comment, so nothing enforces it. ADR-0020's
consequences flag both as not yet centrally tuned, at `docs/adr/0020-…:58`).

## Verifying a trace end to end (not a unit test)

A unit test cannot prove propagation — see the noop-provider warning above. A-04's "done when"
requires a real collector, and **no collector exists anywhere in this repo**: grepping
`docker-compose.yml`, `deploy/` and `.env.example` for `otel` returns nothing, and no file sets
`OTEL_ENABLED`. Standing one up is part of the task; `docker-compose.yml` already uses profiles
(`infra`, `ha`, `e2e`), so a collector service belongs there.

Procedure:

1. Start a collector (or Jaeger/Tempo) exposing OTLP/gRPC on `:4317`.
2. `make build && OTEL_ENABLED=true OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
   OTEL_EXPORTER_OTLP_INSECURE=true OTEL_SERVICE_NAME=fuse ./bin/fuse server -l debug`
3. Trigger with an inbound parent — **this is the break-1 repro; today the header is read by nothing**:
   ```bash
   curl -s -X POST http://localhost:9090/v1/workflows/trigger \
     -H 'Content-Type: application/json' \
     -H 'traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01' \
     -d '{"schemaID":"smallest-test"}'
   ```
4. In the collector, assert one trace containing `workflow.execute` with `node.execute` children — and,
   after A-04, that its trace id is `4bf92f3577b34da6a3ce929d0e0e4736`.

⚠️ The e2e helpers cannot send custom headers: `tests/e2e/http.go` builds the request internally and
takes no header argument — `POSTJSON` (L112) sets only `Content-Type`, `GET` (L94) sets none at all.
An inbound-traceparent e2e test needs a new helper there.

## Traps (all confirmed)

| Trap | Evidence |
| --- | --- |
| **Cancelled workflows leak the span and the gauge and skip the trace.** `handleMsgCancelWorkflow` (L579) sets `StateCancelled` and returns without calling `sendWorkflowCompleted()`, whose only two callers are L480 and L486. So `rootSpan.End()` (L511) never runs, `WorkflowsActive.Dec()` (L498) never runs, the `StateCancelled` branch (L506-508) is unreachable from cancellation so `fuse_workflows_cancelled_total` stays 0, and `persistTrace()` never runs. **Do not fix this inside an A-04 PR** — the backlog forbids drive-by fixes; file it with evidence. | `workflow_handler.go:480, 486, 494-519, 579` |
| **Intercepted `system/*` functions produce no `node.execute` span and no duration metric.** `handleWorkflowRunFunctionAction` returns at L675-688 before dispatch, so sleeps, waits, sub-workflow spawns and foreach fan-outs are invisible in OTel and contribute nothing to `fuse_node_exec_duration_seconds`. | `workflow_handler.go:671-688` |
| **Async node spans measure dispatch, not work.** `nodeSpan.End()` (L166) fires as soon as `ExecuteFunction` returns, and every `ai/chat`, `ai/agent` and `logic/timer` node returns `NewFunctionResultAsync()` immediately. The completion path (`transport/internal.go` `sendAsyncFunctionResult`, and `handlers/async_function_result.go`) carries **no** `TraceCarrier` at all. | `workflow_func.go:166`; `transport/internal.go:18-25` |
| **Replay/recovery mints a second, unrelated trace.** `startRootSpan` always starts from `context.Background()`, so a run recovered after restart (L143) or stolen by the HA claim sweep gets a fresh trace id with no link to the original. No field stores a run's trace id. | `workflow_handler.go:143, 214-216` |
| **`BuildTrace` consumes 5 of 18 journal entry types** — `step:started`, `step:completed`, `step:failed`, `step:retrying`, `state:changed`. Sleeps, awakeables, sub-workflow relationships and foreach structure never appear in the persisted trace. | `internal/workflow/trace_builder.go:22-68` vs the 18 constants in `internal/workflow/journal.go:15-49` |
| **Doubled function id in the concurrency/rate-limit key.** `workflow_func.go:101` builds `fmt.Sprintf("%s/%s", PackageID, FunctionID)` but `FunctionID` is already the full id, yielding `fuse/pkg/logic/fuse/pkg/logic/sum`. It is used only as the bucket key (L103, L109); the Prometheus label (L114, L132, L167) correctly uses the plain `FunctionID`. Mitigating: those metadata fields are never populated, so the branches are unreachable today. **No backlog task owns it** — file it. | `workflow_func.go:101-109` |
| **`LOG_LEVEL` is not read.** `ParamsConfig.LogLevel` has no `env` tag; the level comes only from the cobra `--loglevel`/`-l` flag. `LOG_FORMAT` **is** tagged (default `json`) and `--log-format` overrides it when non-empty. | `internal/app/config/config.go`; `internal/app/cli/root.go` |
| **Log-format ordering hazard.** `internal/logging/app_logger.go` holds a package-global mutable `logFormat`, written only by `NewAppLogger`. `ErgoLogger()` and the `NewFxLogger()` closure both read it through `newLogger()`, and fx builds its logger early — so fx lifecycle lines can be JSON while the rest of the process is console. Do not add a second format switch. | `internal/logging/app_logger.go`, `fx_logger.go`, `ergo_logger.go` |
| **ergo panic-level logs are downgraded on purpose** (`gen.LogLevelPanic` → zerolog Error) because `zerolog.Panic()` re-panics and can crash the node adapter. Do not "restore" it. | `internal/logging/ergo_logger.go` |
| **`.agents/rules/03-actor-patterns.mdc` uses a method that does not exist**: `a.Log().Warn(...)`. The real ergo API is `Warning` — the codebase uses `a.Log().Warning(` and never `a.Log().Warn(`. | grep the tree |
| **There is no `GET /v1/workflows/{workflowID}/status` route**, and no run-state endpoint returning audit data or a `logs` field. The real route is `/v1/workflows/{workflowID}` and its response DTO carries only workflow id and status. | `internal/actors/mux_worker.go`; `internal/dtos/workflow.go` |

Inside actors log via `a.Log()` / `h.Log()`. Code that runs outside an actor — node functions,
goroutines, transports — uses the global zerolog `log` (see `transport/internal.go`, `ai/chat.go`,
`http/request.go`).

## Commands (confirmed to exist)

```bash
make lint && make build && make test        # the mandated gate, in this order
make swagger                                # REQUIRED on a fresh clone before build/test
go test -v ./internal/tracing/...           # 6 provider tests — every one builds from noopConfig() (OTel DISABLED)
go test -v ./internal/metrics/...
go test -v ./internal/logging/...
go test -v -run TestBuildTrace ./internal/workflow/
make build && ./bin/fuse server -l debug --log-format console
curl -s http://localhost:9090/metrics | grep -E '^fuse_|^ergo_'
curl -s http://localhost:9090/v1/workflows/<workflowID>/trace | jq .
curl -s 'http://localhost:9090/v1/schemas/<schemaID>/traces?limit=10' | jq .
make e2e-local                              # builds fuse-app:test, compose --profile e2e, -tags=e2e, down -v
E2E_API_URL=http://localhost:9091 go test -tags=e2e ./tests/e2e -run TestE2E_GET_workflow_trace -v -count=1
```

$ARGUMENTS
