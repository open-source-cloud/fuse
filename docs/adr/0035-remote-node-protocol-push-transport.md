# 0035. Remote node protocol: push-based HTTP as the v1 transport

- Status: Proposed
- Date: 2026-09-10
- Deciders: FUSE maintainers

## Context and Problem Statement

FUSE lets a workflow node reference a registered capability (`pkg/workflow/package.go`), and the
engine already has a transport seam — `internal/packages/transport.FunctionTransport` — with exactly
one implementation, `InternalFunctionTransport`. `pkg/transport/type.go` declares `HTTP` and `gRPC`
constants that no production code reads, and `POST /v1/workflows/{workflowID}/execs/{execID}`
already exists as an unvalidated completion callback. None of this is documented as a contract:
`PackagedFunction.Function` is `json:"-"`, so a capability registered over the REST API today lands
metadata-only and fails to dispatch with `"function %s has no transport"`
(`internal/packages/loaded_package.go:41`). ADR-0024 named this explicitly as a deferred decision:
out-of-process functions "await an HTTP/gRPC transport implementation... a separate future decision."

Before any Go is written for that transport (**B-02**), FUSE needs an explicit, versioned answer to
eight questions a worker SDK author depends on: dispatch direction, invocation envelope, ack
semantics, completion taxonomy, delivery guarantee, auth, capability schema, and versioning. This ADR
records the **why** behind the highest-leverage, hardest-to-reverse of those — dispatch direction —
and points to the full wire-level answer to all eight.

## Decision Drivers

- FUSE's positioning is "register your existing API as a node" — the worker is typically already an
  HTTP server (Laravel, NestJS) before it ever talks to FUSE.
- The only worker-facing inbound surface that exists today is a completion callback
  (`internal/actors/mux_worker.go:79`), not a work-lease/poll/stream endpoint — building on what
  exists is cheaper than inventing a queueing protocol from nothing.
- Dispatch direction is the hardest of the eight decisions to change later: it shapes SDK connection
  lifecycle, every customer firewall rule, and the auth model (both directions) from day one.
- Retry safety, delivery guarantees, and error routing all depend on a completion taxonomy the engine
  does not have yet (`HandleNodeFailure`, `internal/workflow/workflow.go:841`, applies one retry
  policy uniformly today — see ADR-0022).
- Idempotency dedup should live in the SDK, not be re-invented per customer controller — the option
  the protocol chooses must make that cheap.

## Considered Options

- **A — Push.** FUSE calls the worker's registered HTTP endpoint per dispatch; the worker responds
  synchronously or acknowledges `202` and completes later via the existing callback route.
- **B — Pull.** The worker polls or opens a long-lived stream to FUSE for work items; FUSE holds a
  queue per capability.
- **C — Both, worker-selectable per capability, in v1.**

## Decision Outcome

Chosen option: **A — push**, specified in full at
[`docs/protocol/remote-node-protocol-v1.md`](../protocol/remote-node-protocol-v1.md) (v1.0.0-draft,
clause `RNP-5.1`). Pull is reserved as a distinct, additively-versioned transport for a later
protocol revision, for deployments with no inbound reachability to the worker — explicitly out of
scope for v1 and not to be assumed by any v1-conformant worker.

Push is the only option with an existing foothold in the engine (the callback route) and the only
one that matches "an existing Laravel/NestJS API is already an HTTP server" without asking every
worker to also become a queue consumer on day one. Pull is real and eventually necessary (it is how
Temporal survives locked-down enterprise networks), but nothing in `internal/actors/mux_worker.go`'s
route table is a lease/poll/stream endpoint today, so building it alongside push in v1 would mean
designing and shipping two transports before either has a customer. Option C is rejected for the
same reason: it doubles the v1 surface area for a capability class (no-inbound-reachability workers)
this repo has no evidence exists yet among early adopters.

### Consequences

- Good: a worker author with an existing HTTP service registers it and starts receiving dispatches
  with no new inbound infrastructure — the fastest path to the first SDK.
- Good: builds directly on the one inbound primitive that already exists
  (`internal/handlers/async_function_result.go`), rather than inventing a second one from nothing.
- Bad: every customer must allow inbound connectivity from FUSE to the worker — unworkable for
  locked-down enterprise networks until a pull transport ships.
- Bad: this is the single hardest clause in the protocol to reverse; committing to push first means
  a pull transport, when built, must be designed as an additive protocol capability
  (`RNP-12.1`'s version-negotiation mechanism), not a v1 breaking change.
- Neutral: the full wire contract (envelope, ack semantics, completion taxonomy, delivery guarantee,
  auth, capability schema, versioning) is specified separately in
  `docs/protocol/remote-node-protocol-v1.md` rather than duplicated in this ADR — this ADR is the
  record of *why* push, not *what* a worker implements.

## Pros and Cons of the Options

### A — Push (chosen)

- Good: idiomatic for HTTP-framework workers; builds on the existing callback route; fastest to a
  working SDK.
- Bad: requires inbound reachability to the worker; not viable behind a restrictive firewall/NAT
  without a tunnel.

### B — Pull

- Good: works with zero inbound reachability — the enterprise-network-survival property.
- Bad: no existing engine primitive to build on (no lease/poll/stream route exists); requires
  inventing queue semantics, lease timeouts, and visibility windows before a single SDK can ship;
  worse ergonomic fit for "wrap my existing API" framing.

### C — Both, worker-selectable, in v1

- Good: no customer is blocked by network topology from day one.
- Bad: doubles v1 scope and testing surface for a customer segment (no inbound reachability) not yet
  evidenced; delays shipping any SDK while both transports are built and specified.

## More Information

- Full wire specification (all eight B-01 decisions, normative clause ids `RNP-*`):
  [`docs/protocol/remote-node-protocol-v1.md`](../protocol/remote-node-protocol-v1.md).
- Code: `pkg/transport/type.go` (the dead `HTTP`/`gRPC` enum), `internal/packages/transport/`
  (`FunctionTransport`, `InternalFunctionTransport`), `internal/packages/loaded_package.go`
  (registration branch, `MapToRegistryPackage`), `internal/actors/mux_worker.go:79` and
  `internal/handlers/async_function_result.go` (the existing callback route this decision builds on).
- Related: [ADR-0024](0024-package-registry-and-function-metadata.md) (named this transport decision
  as deferred), [ADR-0022](0022-retry-and-error-handling-model.md) (retry/error model the completion
  taxonomy in the spec extends), [ADR-0023](0023-timeout-enforcement-model.md) (timeouts, and its
  caveat that a remote worker keeps running after the engine's own deadline fires — the spec's
  `deadline` field inherits that caveat), [ADR-0017](0017-idempotency-check-and-set.md) (idempotency
  is trigger-level only today — an explicit scope limit the spec's per-exec idempotency key,
  `RNP-9.2`, does not extend past), [ADR-0027](0027-async-tool-invocation-sub-execution-channel.md)
  (forbids adding a per-execution runtime handle back onto `workflow.ExecutionInfo`; the spec's new
  envelope fields are plain data, not a handle, and are constrained accordingly).
- Backlog: `BACKLOG_V2.md` — **B-01** (this decision), **A-03** (replay semantics for pending remote
  steps, which the spec states the outcome of at `RNP-9.1` but which still owns its own acceptance),
  **B-02** (implements the transport this ADR names), **B-03/B-05/B-08/C-10** (all depend on this
  ADR and the linked spec being published).
