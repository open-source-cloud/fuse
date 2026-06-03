# 0033. Dependency injection & application composition with uber-go/fx modules

- Status: Accepted
- Date: 2026-06-03
- Deciders: FUSE maintainers

## Context and Problem Statement

FUSE boots a sizable graph of subsystems — config, structured logging, metrics/tracing, the ergo
node and its actor factories, repositories (memory or Postgres), an object store, services, HTTP
handlers, the secret store, the LLM registry, idempotency, concurrency limiters, the event bus —
with real wiring constraints: a strict construction order, **driver-selected backends** (memory vs
Postgres, fs vs S3), **optional dependencies** (no DB pool under the memory driver), **lifecycle**
(open/close pools, start/stop the node), and the need to build **partial graphs** for CLI
subcommands (`fuse migrate`, `fuse secrets`, `fuse credentials`, `fuse workflow`) and tests. This
records how that composition root is built. ADR-0002 chose ergo for the actor *runtime*; it does
not cover how the application is *assembled*.

## Decision Drivers

- An explicit, inspectable dependency graph instead of hand-ordered constructor calls in `main`.
- First-class **optional** dependencies (e.g. a `*pgxpool.Pool` that is nil under `DB_DRIVER=memory`).
- **Lifecycle** hooks (resource open/close, migrations, node start/stop) tied to app start/stop.
- **Driver selection** at provide time (one provider returns the memory or Postgres implementation).
- **Composability** — boot a subset of modules for a CLI command or a test without the full server.
- A single composition root, keeping wiring out of business logic.

## Considered Options

- **A — Manual constructor wiring in `main`** (plain Go).
- **B — uber-go/fx** (runtime DI container with modules, lifecycle, and parameter/result objects)
  (chosen).
- **C — google/wire** (compile-time DI via code generation).

## Decision Outcome

Chosen: **uber-go/fx.** Each layer is an `fx.Module` — `CommonModule` (config, loggers,
metrics, tracing), `DatabaseModule`, `RepoModule`, `ServicesModule`, `WorkerModule` (HTTP handler
factories), `ActorModule` (ergo actor factories), `SecretsModule`, `LLMModule`, `ObjectStoreModule`,
`IdempotencyModule`, `ConcurrencyModule`, `EventsModule`, `PackageModule`, `FuseAppModule` — composed
into **`AllModules`** (`internal/app/di/di.go`). Conventions:

- **`fx.Provide`** for constructors; **`fx.Invoke`** for side-effecting initialization that must run
  (e.g. forcing logger init, starting the node, registering internal packages).
- **Driver selection inside providers**: e.g. `provideWorkflowRepository` /
  `provideSecretStore` return the Postgres implementation when `DB_DRIVER=postgres` and a pool
  exists, else the memory one (`internal/app/di/repos.go`, `secrets.go`).
- **Optional dependencies via `fx.In` parameter structs with `optional:"true"`** — the
  `*pgxpool.Pool` is provided through an `fx.Out` result and consumed optionally, so memory-driver
  runs and CLI commands work with a nil pool (`internal/app/di/database.go`).
- **Lifecycle via `fx.Lifecycle` hooks** — pool open/close, migrations at provide time, node
  start/stop.
- **fx's own logs routed through zerolog** (`logging.NewFxLogger`) so DI diagnostics match the app
  log format ([ADR-0020](0020-observability-metrics-tracing-execution-traces.md)).
- **Partial graphs**: CLI subcommands assemble only the modules they need (e.g. `CommonModule +
  DatabaseModule + SecretsModule`) via `fx.New(...)`, reusing the same providers as the server.

### Consequences

- Good: the dependency graph is explicit and centralized; adding a subsystem is a new module, not a
  surgical edit to `main`.
- Good: optional/driver/lifecycle patterns are first-class, which the memory-vs-Postgres and CLI
  paths rely on heavily.
- Good: the same providers back the server, the CLI, and tests — no duplicate wiring.
- Bad: wiring errors surface at **runtime** (app start), not compile time; a missing/ambiguous
  provider fails on boot.
- Bad: fx adds reflection and a learning curve; module boundaries must be maintained deliberately.
- Neutral: fx is now a load-bearing, costly-to-reverse framework choice on par with ergo.

## Pros and Cons of the Options

### A — Manual wiring in `main`
- Good: no framework, compile-time safety, zero reflection.
- Bad: verbose and brittle as the graph grows; hand-managed ordering, optional deps, lifecycle, and
  per-driver branching become error-prone; partial graphs for CLI/tests duplicate wiring. Rejected.

### B — uber-go/fx (chosen)
- Good: explicit graph, modules, lifecycle, optional deps, partial graphs, shared providers.
- Bad: runtime (not compile-time) errors; reflection; learning curve.

### C — google/wire
- Good: compile-time, no reflection, generated code is debuggable.
- Bad: codegen step in the build; less ergonomic for runtime driver selection and `optional`
  dependencies, which are pervasive here. Rejected.

## More Information

- Code: `internal/app/di/di.go` (`AllModules`, `CommonModule`, `PackageModule`, `FuseAppModule`,
  `Run`), and the per-layer modules in `internal/app/di/*.go` (`database.go`, `repos.go`,
  `services.go`, `actors.go`, `secrets.go`, `llm.go`, `objectstore.go`, …).
- Related: [ADR-0002](0002-ergo-actor-model-for-workflow-execution.md) (the actor runtime fx wires),
  [ADR-0003](0003-in-memory-repositories-by-default.md) (the memory/Postgres driver selection fx
  expresses), [ADR-0020](0020-observability-metrics-tracing-execution-traces.md) (fx logs via
  zerolog).
