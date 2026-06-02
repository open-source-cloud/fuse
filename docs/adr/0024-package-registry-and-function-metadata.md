# 0024. Function packages: a registry with declarative metadata as contract

- Status: Accepted
- Date: 2026-06-01
- Deciders: FUSE maintainers

> Backfill ADR: records a foundational decision already implemented.

## Context and Problem Statement

A workflow node references "a function" to execute. The engine needs a model for what a function
*is*: how it's discovered, how its inputs/outputs are described well enough to validate schemas,
route conditional edges, and (later) generate LLM tool schemas, and how it's invoked. A bare Go
function pointer carries none of that contract. This model underpins nodes, the schema layer
([ADR-0013](0013-workflow-schema-model-and-input-mapping.md)), conditional routing
([ADR-0015](0015-conditional-routing-with-expr-lang.md)), and agent tools
([ADR-0007](0007-agent-reasoning-loop-and-tools-from-functions.md)).

## Decision Drivers

- Describe a function's typed inputs/outputs declaratively (validate before running).
- Discover functions by id (`package/function`) and manage them at runtime via the API.
- Carry per-function execution attributes (transport, concurrency, rate limit, conditional edges).
- One contract reusable across nodes, the REST API, and agent tool generation.

## Considered Options

- **First-class `Package`/`PackagedFunction` units with declarative `FunctionMetadata`, held in a
  registry.**
- **Raw Go function pointers** referenced directly, with shapes known only at call time.
- **An external RPC/plugin contract** (every function is an out-of-process service).

## Decision Outcome

Chosen option: **declarative, metadata-described functions in a registry.** A `Package`
(`pkg/workflow/package.go`) groups `PackagedFunction`s, each pairing an executable `Function` with
`FunctionMetadata` (`pkg/workflow/metadata.go`): `InputMetadata`/`OutputMetadata` made of
`ParameterSchema{Name, Type, Required, Validations, Description, Default}`, plus conditional-edge
metadata, transport, and concurrency/rate-limit config. A `Registry`
(`internal/packages/registry.go`, `MemoryRegistry`: `Register`/`Get`/`Has`/`List`) holds
`LoadedPackage`s; `LoadedPackage.ExecuteFunction` runs a function through its transport
([ADR transport seam] — `internal/packages/transport`). Built-in packages register at startup
(`internal_packages.go` `RegisterInternalPackages`); packages are introspectable/manageable over
`GET /v1/packages` and `PUT/GET /v1/packages/{id}`.

This metadata is the single typed contract reused everywhere: nodes validate and coerce inputs
against it, conditional output fields/edges are declared in it, and agent tool schemas are derived
from `Input.Parameters` ([ADR-0007](0007-agent-reasoning-loop-and-tools-from-functions.md)).

### Consequences

- Good: schemas can be validated before execution (do inputs match? is this edge valid?); one
  contract powers nodes, the API, and agent tools; functions are discoverable and self-describing.
- Good: per-function execution policy (concurrency/rate limit — [ADR-0016](0016-concurrency-and-rate-limiting.md))
  travels with the metadata.
- Bad: authoring a function means authoring its metadata too (more boilerplate than a bare func).
- Neutral: today functions register **in-process** at startup; out-of-process/remote functions
  await an HTTP/gRPC transport implementation (the transport interface exists with `Internal` built
  and `HTTP`/`gRPC` stubbed — a separate future decision).

## Pros and Cons of the Options

### Registry + declarative metadata (chosen)

- Good: validation, introspection, one reusable contract, per-function policy.
- Bad: metadata boilerplate per function.

### Raw Go function pointers

- Good: minimal; no metadata to write.
- Bad: no typed contract — can't validate schemas/edges or generate tool schemas; not manageable
  over an API.

### External RPC/plugin contract for all functions

- Good: language-agnostic, strong isolation.
- Bad: heavy for built-ins; network hop and operational cost for every call; overkill now (kept as
  a future option behind the transport seam).

## More Information

- Code: `pkg/workflow/package.go` (`Package`, `PackagedFunction`, `NewPackage`/`NewFunction`),
  `pkg/workflow/metadata.go` (`FunctionMetadata`, `ParameterSchema`);
  `internal/packages/registry.go`, `loaded_package.go`, `internal_packages.go`;
  `internal/handlers/packages.go` (REST API); transport in `internal/packages/transport/`.
- Related: [ADR-0013](0013-workflow-schema-model-and-input-mapping.md),
  [ADR-0015](0015-conditional-routing-with-expr-lang.md),
  [ADR-0007](0007-agent-reasoning-loop-and-tools-from-functions.md),
  [ADR-0016](0016-concurrency-and-rate-limiting.md).
