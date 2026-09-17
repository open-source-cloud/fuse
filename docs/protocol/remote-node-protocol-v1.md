# FUSE Remote Node Protocol — v1.0.0-draft

Status: **Draft**. This document specifies a protocol that **does not yet execute in the engine**.
It is the contract B-02 (`internal/packages/transport`, a new remote transport) must implement, and
the contract an SDK author (Laravel, NestJS, or any HTTP-capable worker) implements against. Every
clause below is either marked `NORMATIVE` (binding on the engine, a worker, or both — a B-05
conformance test may cite it by clause id) or `INFORMATIVE` (context, rationale, non-binding).

Companion decision record: [ADR-0035 — Push-based HTTP as the v1 remote node transport](../adr/0035-remote-node-protocol-push-transport.md).

## 0. Changelog

There is no CHANGELOG file in this repository (flagged as a gap; not created here — out of scope
for B-01, which authors protocol content, not repo tooling). Revisions to this document are tracked
in this section until such a file exists.

| Version | Date | Change |
| --- | --- | --- |
| v1.0.0-draft | 2026-09-10 | Initial draft. Settles all eight B-01 decisions. No SDK exists yet; no clause is binding on any shipped code. |

**Change-control rule (`NORMATIVE`, clause `RNP-0.1`):** Once this document reaches `v1.0.0`
(non-draft) and at least one SDK implements it, no clause may be edited in place. A behavioural
change to a published clause MUST be a new minor/major protocol version (§12) with a migration note
appended to this changelog, or — if urgent and narrow — an errata entry that does not change
wire behaviour. Editing a published clause's normative meaning without a version bump is a protocol
break, not a documentation fix.

---

## 1. Scope and audience

This document specifies the wire contract between the FUSE engine ("FUSE", "the engine") and an
out-of-process **remote node worker** ("the worker", "the SDK") that implements one or more
**capabilities** (functions) FUSE can dispatch to as part of a workflow node. It is written to be
implementable by someone who has read this document and never opened a `.go` file.

**Out of scope:**
- The agent execution model (tool-calling loops, orchestrator mode) — gated on F-02 per the backlog;
  not designed here. Where a remote capability interacts with the agent/tool-projection surface
  (§11.4), this document states only what is true today and what is explicitly deferred.
- The TypeScript and PHP SDK implementations themselves (B-06/B-07) — they consume this document,
  they are not specified by it.
- The capability registry's storage/API model (B-03) beyond the wire shape a remote capability's
  declaration MUST have (§11).
- Worker lifecycle (registration health, deregistration, draining, multi-worker addressing) — B-08's
  subject; §13 states only the parts this protocol version already constrains.

## 2. Conformance language

MUST / MUST NOT / SHOULD / SHOULD NOT / MAY are used per RFC 2119. Every normative clause has a
stable id of the form `RNP-<section>.<n>` (Remote Node Protocol). A future B-05 conformance suite
cites these ids directly; do not renumber a published clause — see `RNP-0.1`.

## 3. Reading this document against the engine

Every engine claim in this document is one of exactly two kinds, and they are never mixed within a
sentence:

- **`§ TODAY`** — shipped behaviour, cited `file:symbol` or `file:line`, personally confirmed by
  reading the file (Appendix A is the full inventory).
- **`§ PROTOCOL`** — a requirement this document imposes that the engine and/or worker MUST grow to
  become conformant. Nothing marked `§ PROTOCOL` exists in `main` at the time of writing. Appendix B
  lists every such gap with the backlog id that owns building it.

If a sentence below is not explicitly tagged, its tag is inherited from the nearest heading.

## 4. Terminology

| Term | Meaning |
| --- | --- |
| **Capability** | A registered, invokable function a graph node can reference: `<packageId>/<functionId>`. |
| **Worker** | An out-of-process HTTP server implementing one or more capabilities per this protocol. |
| **Dispatch** | FUSE's outbound HTTP call to a worker invoking one capability for one execution. |
| **Exec** | One execution attempt of one node, identified by `execId` (`workflow.ExecID`, §6). |
| **Callback** | The worker's inbound `POST` completing an async dispatch. |
| **Envelope** | The JSON body FUSE sends on dispatch (§6). |

---

## 5. Decision 1 — Push or pull

**Decision (`NORMATIVE`, `RNP-5.1`):** FUSE dispatches by calling the worker's HTTP endpoint
("push"). A pull/poll/lease transport MAY be added as a second, independently-versioned transport in
a future protocol revision for network topologies with no inbound reachability to the worker; it is
explicitly **out of scope for v1** and MUST NOT be assumed by any v1-conformant worker.

**Rejected alternative:** worker-side polling or a long-lived stream (gRPC or SSE) from the worker to
FUSE, pulling work items off a queue.

**Why (`INFORMATIVE`):** Push matches "register your existing API as a node" for NestJS and Laravel
— both are already long-running HTTP servers, so no new inbound listener, poll loop, or persistent
connection is required to receive work. Pull is the correct answer for a worker with no inbound
reachability (locked-down enterprise networks — the reason Temporal's SDKs default to pull), but
nothing in the engine today gives it anywhere to land.

**Engine constraint (`§ TODAY`):** the complete route table in `internal/actors/mux_worker.go` (the
`Routes` slice built at lines ~46–140+) contains no lease, poll, or stream endpoint. The only
worker-facing *inbound* surface anywhere in the tree is the async-result callback,
`POST /v1/workflows/{workflowID}/execs/{execID}` (`internal/actors/mux_worker.go:79`,
`internal/handlers/async_function_result.go:53`) — and that is a completion callback, not a work
lease; there is nothing a worker can call to be handed a job. Other inbound routes exist (webhook
triggers, `POST /v1/awakeables/{awakeableID}/resolve` at `internal/handlers/resolve_awakeable.go:55`)
but none hands work *out* to a worker.

**Cost to reverse (`INFORMATIVE`):** highest in this document. Push vs. pull shapes SDK connection
lifecycle, customer firewall rules, and the auth model (§10) from day one. Adding pull later as
protocol version `2.x` (§12) is additive — it does not require breaking push — but retrofitting pull
semantics onto a push-shaped v1 SDK would.

---

## 6. Decision 2 — Invocation envelope

**Decision (`NORMATIVE`, `RNP-6.1`):** FUSE dispatches a capability with an HTTP `POST` to the
worker's registered endpoint for that capability, with this JSON body ("the envelope"):

```jsonc
{
  "protocolVersion": "1.0",          // RNP-6.2  — see §12
  "workflowId": "wf_...",            // RNP-6.3  — § TODAY, carried today
  "execId": "018f...",               // RNP-6.4  — § TODAY, carried today
  "nodeId": "http-request-1",        // RNP-6.5  — § PROTOCOL, see gap below
  "threadId": 0,                     // RNP-6.6  — § TODAY, carried but not threaded to the function today
  "attempt": 0,                      // RNP-6.7  — § PROTOCOL, see gap below (0-indexed)
  "idempotencyKey": "018f....0",     // RNP-6.8  — § PROTOCOL, see §9
  "deadline": "2026-09-10T12:00:30Z",// RNP-6.9  — § PROTOCOL, see gap below
  "environment": "production",       // RNP-6.10 — § TODAY, carried today
  "functionId": "myco/billing/charge", // RNP-6.11 — § TODAY, carried today (ExecuteFunctionMessage.FunctionID)
  "input": { "...": "..." },         // RNP-6.12 — § TODAY, carried today
  "callbackUrl": "https://engine.example.com/v1/workflows/wf_.../execs/018f...", // RNP-6.13 — § PROTOCOL
  "callbackToken": "cbt_...",        // RNP-6.14 — § PROTOCOL, see §10.2
  "traceparent": "00-4bf9...-1"      // RNP-6.15 — § PROTOCOL, see gap below
}
```

Field-by-field engine status:

| Field | Type | Today? | Evidence / gap |
| --- | --- | --- | --- |
| `protocolVersion` | string | **Gap** | No representation anywhere. New field, see §12. |
| `workflowId` | string | **Carried** | `ExecuteFunctionMessage.WorkflowID` (`internal/messaging/execute_function.go:13`) → `ExecutionInfo.WorkflowID` (`pkg/workflow/execution_info.go:15`). |
| `execId` | string | **Carried** | `ExecuteFunctionMessage.ExecID` (`:14`) → `ExecutionInfo.ExecID` (`:16`). |
| `nodeId` | string | **Gap** | Exists only as `entry.FunctionNodeID` on audit-log/journal entries (`internal/workflow/workflow.go:428`, `:437`), never on the dispatch path. Not on `ExecuteFunctionMessage`, not on `ExecutionInfo`. |
| `threadId` | uint16 | **Partially carried** | On `ExecuteFunctionMessage.ThreadID` (`:15`), but `ExecutionInfo` has no field for it — it does not reach the function today. Also recoverable as `ExecID.Thread()` (`pkg/workflow/exec_id.go:23`), which decodes bytes 6–7 of the UUID and **discards the hex-decode error** (`:27`) — a worker MUST NOT rely on independently re-deriving `threadId` from `execId`; FUSE MUST send it explicitly. |
| `attempt` | int | **Gap** | Exists as `workflowactions.RetryFunctionAction.Attempt` (`internal/workflow/workflowactions/action.go:55`), set in `HandleNodeFailure`, but never threaded into `ExecuteFunctionMessage` or `ExecutionInfo`. This is backlog item **A-08**; §9 states how `attempt` and the idempotency key interact until A-08 ships. |
| `idempotencyKey` | string | **Gap** | No producer anywhere derives a key from `(workflowId, execId, attempt)`. See §9. |
| `deadline` | RFC 3339 timestamp | **Gap** | The engine has a per-node `TimeoutConfig.Execution` (`internal/workflow/timeout.go`, ADR-0023) enforced by an `ExecutionTimer` (`internal/actors/execution_timer.go`), but that deadline is never surfaced to the function or the envelope. §7.4 states the semantics once it is. |
| `environment` | string | **Carried** | `ExecuteFunctionMessage.Environment` (`:19`) → `ExecutionInfo.Environment` (`:20`) — ADR-0031 resolution scope. |
| `functionId` | string | **Carried** | `ExecuteFunctionMessage.FunctionID` (`:17`). |
| `input` | object | **Carried** | `ExecuteFunctionMessage.Input` (`:18`) → `ExecutionInfo.Input` (`:21`). |
| `callbackUrl` | string (absolute URL) | **Gap** | Nothing constructs an absolute externally-reachable URL today. `internal/app/config/config.go:68`'s `BaseURL` is per-LLM-provider config (`LLMProviderConfig.BaseURL`), unrelated. A server-identity base URL (e.g. `PUBLIC_BASE_URL`) does not exist in `Config` and MUST be added for B-02 to construct this field. |
| `callbackToken` | string | **Gap** | New field, see §10.2. |
| `traceparent` | W3C trace-context string | **Carried one level up, not on the envelope path** | `messaging.Message.TraceCarrier map[string]string` (`internal/messaging/message.go:44`) exists and is populated by `NewExecuteFunctionMessage`'s `traceCarrier` parameter (`execute_function.go:26`), but `ExecutionInfo` has no `context.Context` or carrier field, so nothing propagates it to a function body — this is backlog **A-04**, called out explicitly as the blocking dependency of B-02 in `BACKLOG_V2.md:113`. |

**Engine constraint on the closest existing analogue (`§ TODAY`, `RNP-6.16`):**
`messaging.ExecuteFunctionMessage` (`internal/messaging/execute_function.go:12-20`) is the nearest
thing to an invocation envelope that exists, and it carries exactly seven fields: `WorkflowID`,
`ExecID`, `ThreadID`, `PackageID`, `FunctionID`, `Input`, `Environment`. Of those seven, only four
(`WorkflowID`, `ExecID`, `Input`, `Environment`) survive into `ExecutionInfo`
(`pkg/workflow/execution_info.go:14-23`), the struct an executable function actually receives. This
spec's envelope is a superset built for the wire, not a description of `ExecutionInfo` as it stands.

**Rejected alternative:** re-using `ExecuteFunctionMessage`'s JSON tags verbatim as the wire shape.
Rejected because it silently omits `nodeId`, `attempt`, `deadline`, `idempotencyKey`, `callbackUrl`
and `traceparent` — the fields that make retries, dedup, and tracing possible across a network
boundary — and because `ExecuteFunctionMessage` is an internal actor-message type, not a published
contract; coupling the wire format to it would make an internal refactor a protocol break.

**Cost to reverse (`INFORMATIVE`):** medium. Additive fields are cheap for a v1.x revision (§12);
renaming or repurposing a field already consumed by a shipped SDK is not — see `RNP-0.1`.

---

## 7. Decision 3 — Ack semantics

**Decision (`NORMATIVE`):**

- `RNP-7.1` — On receiving a dispatch, the worker MUST respond either **synchronously** (§7.2) or
  **asynchronously** (§7.3). There is no third mode in v1.
- `RNP-7.2` (sync) — The worker MAY respond `200 OK` with a JSON body
  `{"status": "success"|"error", "errorType"?: "business"|"infrastructure", "data": {...}}`
  (§8 defines the taxonomy) within the dispatch's HTTP timeout. This maps to
  `FunctionResult{Async: false}` (`pkg/workflow/fn_result.go:4-20`).
- `RNP-7.3` (async) — The worker MUST respond `202 Accepted` with an empty or minimal
  acknowledgement body (e.g. `{"accepted": true}`) **immediately**, before doing the work, and
  complete the exec later via the callback (§7.4). This maps to `FunctionResult{Async: true}`,
  produced today in-process by `NewFunctionResultAsync()` (`pkg/workflow/fn_result.go:44-49`), whose
  `Async` bit is the **only** signal the engine currently has for "this will complete later."
- `RNP-7.4` (callback) — The callback is `POST {callbackUrl}` with body
  `{"result": {"status": "success"|"error", "errorType"?: "business"|"infrastructure", "data": {...}}}`
  — the same `dtos.AsyncFunctionRequest{Result workflow.FunctionOutput}` shape that exists today
  (`internal/dtos/workflow.go:29`, route `internal/actors/mux_worker.go:79`), extended with the
  `errorType` field required by §8. FUSE responds `200 {"code":"OK", "workflowId", "execId"}` on
  success and MUST respond with an error status (§7.5) on a request it rejects.
- `RNP-7.5` — If the callback request itself fails (network error, non-2xx from FUSE, timeout), the
  worker MUST retry the callback with exponential backoff (recommended: base 1s, cap 30s, matching
  the engine's own default backoff shape in `internal/workflow/retry.go`, ADR-0022) for at least as
  long as it is willing to hold the result, and MUST treat a `409`/`404` response (§7.6) as a
  terminal, non-retryable outcome — the exec is no longer waiting for it.
- `RNP-7.6` (`§ PROTOCOL`, gap) — FUSE MUST validate, on every callback, that: the workflow exists,
  the exec is genuinely pending, the exec belongs to the stated workflow, and no result was already
  submitted for it — returning `404` for the first two and `409` for the last. **None of this exists
  today.** The current handler (`internal/handlers/async_function_result.go:53-85`) reads two path
  params, binds JSON, and `Send`s unconditionally; it returns `200` whenever the actor `Send`
  succeeds, regardless of whether the workflow handler exists (`tests/e2e/apis_e2e_test.go:201-217`
  documents this as acceptable-for-smoke today, which is precisely the defect this clause closes).
  The validation template to copy is `internal/handlers/resolve_awakeable.go:66-73`, which already
  404s on missing and 400s on non-pending for the awakeable-resolve route. This is **B-02's**
  responsibility to build, not this document's.
- `RNP-7.7` (`§ PROTOCOL`, gap, informative caveat) — **A duplicate callback is not safe today.**
  `Workflow.SetResultFor` (`internal/workflow/workflow.go:422`) overwrites `entry.Result`, re-sets
  `aggregatedOutput`, and appends a **second** `step:completed` journal entry;
  `handleMsgAsyncFunctionResult` then calls `Workflow.Next` again. Its only guard is
  `a.isTerminalState()` (`internal/actors/workflow_handler.go:319`). `RNP-7.6`'s duplicate-`409`
  check is the fix; until B-02 ships it, a worker that retries a callback after a `200` was actually
  received (but the retry response was lost) can advance the graph twice.

**Engine constraint (`§ TODAY`):** the one bit the engine has today is `FunctionResult.Async`
(`pkg/workflow/fn_result.go:4-7`). Everything else in this section (validation, the `errorType`
field, duplicate rejection) is new.

**Rejected alternative:** a single synchronous-only ack (no `202`/callback path at all). Rejected
because it is the reason this protocol exists — a worker calling a slow downstream system (or
performing a long-running task) cannot hold an HTTP connection open for the dispatch's duration
without also holding a `WorkflowFunc` pool worker for that whole time (`PoolSize: 3` per instance,
`internal/actors/workflow_func_pool.go:40` — see backlog **A-10**). The async path is `BACKLOG_V2.md`'s
own framing: "the async path is the interesting one and the reason this exists."

---

## 8. Decision 4 — Completion taxonomy

**Decision (`NORMATIVE`, `RNP-8.1`):** A completed dispatch (sync response or async callback) MUST
report exactly one of three outcomes:

1. **`success`** — the node completed; routes to the node's normal output edge(s).
2. **`error` with `errorType: "business"`** — a logical/domain failure (e.g. "invoice already
   voided"). MUST NOT be retried by the engine's `RetryPolicy`; MUST route directly to the node's
   `EdgeSchema.OnError` edge, per the existing error-edge mechanism (`internal/workflow/workflow.go:899
   findErrorEdges`, ADR-0022) — but skipping the retry loop, not after exhausting it.
3. **`error` with `errorType: "infrastructure"`** — a transient/operational failure (timeout,
   connection refused, 5xx from a downstream the worker called). MUST be retried per the node's
   `RetryPolicy` (`internal/workflow/retry.go`, ADR-0022 — default 3 attempts, exponential,
   1s→30s, ×2), falling through to `OnError` only once retries are exhausted, exactly as today.

`RNP-8.2` — `errorType` MUST be present whenever `status == "error"`. A worker that omits it is
non-conformant. FUSE's B-02 implementation SHOULD default a missing `errorType` to
`"infrastructure"` (fail toward retrying, the conservative default) rather than reject the callback,
to keep a partially-conformant worker's workflow from getting permanently stuck — but SHOULD log a
conformance warning when it does.

**New wire field required (`§ PROTOCOL`, `RNP-8.3`):** **yes.** The engine has no representation of
this taxonomy today. `workflow.FunctionOutputStatus` is a three-value enum
(`pkg/workflow/fn_output.go:6-10`) of which production code branches on exactly two:
`FunctionSuccess` and `FunctionError`. `FunctionError`'s payload is a free-form
`Data["error"]` string (built by `NewFunctionResultError`, `pkg/workflow/fn_result.go:36-41`), and
**every** failure source that exists today — a genuine function error, an execution timeout
(`internal/actors/workflow_handler.go:551,564`, synthesising `Data{"error": "execution timeout
exceeded"}`), and "no transport" (`internal/packages/loaded_package.go:41`) — produces the identical
shape. `HandleNodeFailure` (`internal/workflow/workflow.go:841`) therefore applies **one** retry
policy to all of them; there is no way today to say "don't retry this, it's a business error."
`errorType` is a genuinely new field on `FunctionOutput.Data` (or a sibling top-level field — an
implementation choice for B-02, not fixed by this spec) that the engine must parse and act on.

**Rejected alternative:** a naming convention inside `Data["error"]` (e.g. a magic string prefix).
Rejected explicitly — `BACKLOG_V2.md:183`: "getting this taxonomy right is what makes retries safe,"
and a string convention is exactly the kind of thing a worker can typo past silently. A structured
field with an enum FUSE validates is the only version of this that fails loudly instead of quietly
misrouting.

**Cost to reverse (`INFORMATIVE`):** high — retry safety is the reason this decision exists at all,
per the backlog's own framing.

---

## 9. Decision 5 — Delivery guarantee

**Decision (`NORMATIVE`, `RNP-9.1`):** **(a) at-least-once delivery, with a stable idempotency key
the worker/SDK deduplicates on** — the A-03 option this spec is required to state plainly
(`BACKLOG_V2.md:104`, option (a)).

**Idempotency key derivation (`RNP-9.2`):**

```
idempotencyKey = "<execId>:<attempt>"
```

where `execId` is `workflow.ExecID` (`pkg/workflow/exec_id.go`) and `attempt` is the 0-indexed
retry-attempt counter from §6's envelope. A worker/SDK MUST treat two dispatches carrying the same
`idempotencyKey` as the same logical attempt: if it has already produced (or is producing) a result
for that key, it MUST return the cached result rather than re-executing side effects, and MUST NOT
assume a repeated dispatch with the same key is a new unit of work. This is the SDK-level dedup the
backlog anticipates being "far more palatable... than when the caller was hand-written"
(`BACKLOG_V2.md:106`) — the engine does not need its own dedup store for this; the key just needs to
be derivable and stable.

**Why `execId` alone is not sufficient (`§ TODAY`, `RNP-9.3`):** automatic retries dispatched by
`HandleNodeFailure` **reuse the same `ExecID`** (`RetryFunctionAction{RunFunctionAction{FunctionExecID:
execID}}` — see ADR-0022's "More Information" and `BACKLOG_V2.md:137`), so two different attempts of
the same failed node arrive with an identical `execId`. Only manual `RetryNode` mints a fresh
`ExecID` (`internal/workflow/workflow.go:451`, recording a `previousExecId`). `execId` alone
therefore does not identify an attempt; `attempt` is required in the key, per `RNP-6.7`.

**Owner of dedup (`RNP-9.4`):** the worker/SDK, not the engine. FUSE's obligation is limited to: (1)
deriving `idempotencyKey` deterministically per `RNP-9.2` and including it on every dispatch and
retry dispatch, and (2) not silently changing `execId` on an automatic retry (already true today,
`RNP-9.3`). The SDK's obligation is to key its own dedup store (in-memory, Redis, whatever the
framework provides) on `idempotencyKey` and return the prior result for a repeat.

**Rejected alternative:** (b) replay-aware re-arming of the original `execId` — detecting on restart
that a `step:started` with no completion represents an in-flight remote call, and re-establishing
the original callback binding instead of re-dispatching. Rejected for v1 because it requires the
engine to durably track "this exec is parked awaiting an external callback" in a way that survives
restart, and the current journal already works against that: `handleMsgFunctionResult` calls
`SetResultFor` **before** checking `fnResultMsg.Result.Async`
(`internal/actors/workflow_handler.go:275` vs. `:277`), so the moment a node returns
`NewFunctionResultAsync()`, the journal already carries a `step:completed` for it with placeholder
empty output. `findPendingThreads` (`internal/workflow/workflow.go:267`) matches `step:started`
**without** completion — so it will not see an in-flight remote call as pending at all, and (b) as
originally imagined does not have a foothold to re-arm from without first fixing that trap. Choosing
(a) sidesteps the need to solve that before B-01 can ship; **A-03 is still the task that must record
this choice formally as its own ADR** (`BACKLOG_V2.md:104`: "write it in the protocol spec, because
every SDK author depends on it" — this section is that record; A-03's ADR should reference `RNP-9.1`
rather than re-litigate it).

**Engine constraint forcing this (`§ TODAY`, `RNP-9.5`):** attempt count today lives only in the
in-memory `RetryTracker` (`internal/workflow/retry_tracker.go`), never journalled as a number — this
is backlog **A-08**. Until A-08 ships, `attempt` in the envelope (`RNP-6.7`) MUST still be present
but its value is not guaranteed to reflect a durable count across a restart; a worker SHOULD NOT
treat `attempt` as a globally unique monotonic counter beyond "distinguishes concurrent/retried
dispatches for the same `execId` within one engine process lifetime" until A-08 lands.

**Cost to reverse (`INFORMATIVE`):** high — SDK dedup logic is directly downstream of this key shape.

---

## 10. Decision 6 — Auth, both directions

**Greenfield (`§ TODAY`, `RNP-10.0`):** `docs/API.md:13` states verbatim "**Authentication** — Not
required today; may change in a future version." There is no auth middleware anywhere in
`internal/actors/mux_server.go`, and `Handler.SendJSON` sets `Access-Control-Allow-Origin: *`
(`internal/handlers/handler.go:59`). This section is a 100% new design; nothing below has a shipped
counterpart to contradict.

**Decision (`NORMATIVE`):**

- `RNP-10.1` (FUSE → worker: FUSE proves it is FUSE) — Every dispatch MUST carry
  `Authorization: Bearer <workerAuthToken>`, where `workerAuthToken` is a static, per-capability
  secret established at worker registration and stored via the existing `pkg/secrets` seam:
  `CredentialSecretName(capabilityId, "authToken")` resolves to `cred/<capabilityId>/authToken`
  (`pkg/secrets/credential.go:15`), redacted everywhere except at the point of use via `Reveal()`
  (`pkg/secrets/value.go:33`). Rotating the token means re-registering the capability with a new
  secret value; there is no separate rotation endpoint in v1.
- `RNP-10.2` (worker → FUSE: the worker proves the callback is genuine, and FUSE proves the
  *callback URL* it handed out is legitimate) — the envelope's `callbackToken` (`RNP-6.14`) is a
  single-exec-scoped, engine-minted opaque token, valid only for the specific `(workflowId, execId,
  attempt)` it was issued for. The worker MUST echo it back as
  `Authorization: Bearer <callbackToken>` on the callback request (`RNP-7.4`). FUSE MUST reject a
  callback whose token does not match the pending exec's issued token with `401`, in addition to the
  pending/duplicate checks of `RNP-7.6`. This is a second, independent authentication factor from
  `RNP-10.1` — it does not require the worker's static `workerAuthToken` to reach the callback path
  at all, so a callback token leaked from a log cannot be used to dispatch new work, and a leaked
  `workerAuthToken` cannot be used to forge a completion for an exec it was never handed.
- `RNP-10.3` (replay protection) — FUSE-issued `callbackToken`s MUST be single-use: the first
  callback that presents a valid token consumes it (ties into `RNP-7.6`'s duplicate-`409`); a second
  presentation is rejected as `409`, not re-validated as a fresh callback.
- `RNP-10.4` (clock-skew tolerance) — where a signature or token embeds a timestamp (an
  implementation MAY additionally HMAC-sign the dispatch body for integrity, using the same
  `workerAuthToken` as the HMAC key, header shape `X-Fuse-Timestamp` + `X-Fuse-Signature:
  v1=<hex hmac-sha256>`), the verifier MUST accept a timestamp within **±5 minutes** of its own
  clock and MUST reject anything outside that window. This document does not mandate HMAC signing in
  v1 (`Bearer` alone is the MUST; HMAC is a SHOULD for defense-in-depth) so that a first SDK
  implementation is not blocked on signing-library parity across Laravel/NestJS, but the clock-skew
  figure is fixed here so it does not drift between implementations that do add it.
- `RNP-10.5` (storage) — Both `workerAuthToken` and any per-worker HMAC signing secret MUST be stored
  exclusively via `pkg/secrets` (`cred/<capabilityId>/<field>`), never in `workflow.Package` or any
  DTO that round-trips through `GET /v1/packages` (`workflow.Credential` already carries field
  **names** only, never values — the same discipline applies here).

**Rejected alternatives:** (1) mTLS — rejected for v1 because it requires certificate provisioning
infrastructure this repo has none of, and is a poor fit for "register your existing Laravel/NestJS
API as a node" (most customer HTTP frameworks do not make client-cert verification trivial to bolt
onto an existing route). (2) No auth (ship the greenfield gap forward) — rejected because
`BACKLOG_V2.md:185` is explicit that "customers will run this across trust boundaries," and shipping
an unauthenticated bidirectional callback surface on top of the *already* unvalidated
`async_function_result` route (`RNP-7.6`/`RNP-7.7`) compounds a known defect rather than fixing it.

**Cost to reverse (`INFORMATIVE`):** high — this crosses a trust boundary; changing the scheme after
an SDK ships means coordinating a credential-rotation migration across every customer deployment.

---

## 11. Decision 7 — Capability declaration and schema

**Decision (`NORMATIVE`, `RNP-11.1`):** A capability declares its input and output shape using
**exactly** the parameter type set `internal/typeschema/parse.go:20 ParseValue` accepts:
`string`, `int`, `float64`, `bool`, `[]byte`, `map[string]any`, and `[]T` for any `T` in that set.
A declared `type` string outside this set is non-conformant and MUST be rejected at registration
time (`§ PROTOCOL` — no registration-time schema validation exists today; see Appendix B).

**Engine constraint (`§ TODAY`, `RNP-11.2`):** two shipped internal packages already violate this —
`headers` (`internal/packages/functions/http/request.go:66-67`) and `data`
(`internal/packages/functions/system/wait.go:23`) both declare `Type: "map"`, which `ParseValue`
rejects with `unsupported type: map`. This is a pre-existing engine defect, not something this
document fixes; it is flagged here because a new registration-time validator (`RNP-11.1`) would
reject those two declarations today if applied retroactively to internal packages, and B-02/B-03
implementers MUST NOT let that validator run against already-registered internal packages without
first filing and fixing that defect separately.

**Rejected alternative:** letting a capability declare an arbitrary/open type string and deferring
all validation to runtime coercion failure. Rejected because `applyFlowMapping`
(`internal/workflow/workflow.go:727`) and `inputMapping` (`:652`) already swallow every coercion
failure into a log line followed by `continue`/`break`/`return` — **no error reaches the caller** on
a type mismatch; it silently becomes a missing parameter instead of a failed node
(`internal/workflow/workflow.go:652-751`, confirmed by reading). A capability schema that is not
validated at registration time buys nothing beyond what a worker's own runtime already gives it; the
entire reason to declare a schema is so `applyFlowMapping`'s coercion still has type information to
coerce *against* before that swallow — not to add a second silent-swallow surface on the remote side.

**Edge metadata (`RNP-11.3`):** a registered remote capability MUST be able to declare conditional
input/output edges — `InputEdgeMetadata{Count, Parameters}` and `[]OutputEdgeMetadata{Name,
ConditionalEdge, Count, Parameters}`, matching `pkg/workflow/metadata.go:14-33`'s existing in-process
shape — because a remote node that cannot declare an `OnError` edge cannot participate in the
ADR-0022 failure model at all (§8 depends on it existing somewhere to route to).

**Engine constraint (`§ TODAY`, `RNP-11.4`, gap):** `MapToRegistryPackage`
(`internal/packages/loaded_package.go:63-131`) *does* map `Edges` on both `InputMetadata` and
`OutputMetadata` when building the registry-side struct — confirmed by reading lines 74-77 and
99-110 — so the in-process, code-registered path is not the gap. The gap is the **REST DTO**:
`InputMetadataDTO` (`internal/dtos/package.go:42-45`) has fields `CustomParameters` and `Parameters`
only, and `OutputMetadataDTO` (`:58-62`) has `Parameters`, `ConditionalOutput`,
`ConditionalOutputField` only — **neither DTO has an `Edges` field**, confirmed by reading. A
capability registered via `PUT /v1/packages/{packageID}` (the only registration path available to a
worker with no Go code in this repo) therefore cannot declare edges at all today, and a `GET → PUT`
round-trip of an existing package silently drops any it had. This is the single most direct blocker
to remote capabilities being able to use error-edge routing (§8) and MUST be closed before B-02 is
usable end-to-end — filed as a B-01-owned gap in Appendix B, home is B-02/B-03 per the
`remote-node-protocol` skill's own inference (no backlog task names this defect today).

**Node/tool-projection interaction (`RNP-11.5`, `INFORMATIVE`, capability-registry-and-agents skill):**
capability declaration itself is transport-agnostic — the schema shape in this section does not
differ between an internal and a remote capability. What differs is eligibility for two *other*
surfaces this document does not control: (1) the registry's executable/metadata-only branch
(`internal/packages/loaded_package.go:112-128`), which a remote transport must resolve to an
executable (non-nil `Transport`) entry — B-02's concern, not this spec's; and (2) agent-tool
exposure, gated by `isExposableTool` (`internal/packages/agent_tools.go:95`, requiring `fn.Transport
!= nil && fn.Metadata.Transport == transport.Internal`) — since `MapToRegistryPackage` hardcodes
`Transport: transport.Internal` into every registry entry regardless of wire declaration
(`loaded_package.go:70`), the real discriminator today is the nil-transport check, and **a remote,
async-by-default capability is not eligible as a synchronous agent tool until ADR-0027's async
`ExecRuntime` port lands** — this document does not change that; it is B-03/C-01 territory.

**Concurrency/rate-limit (`RNP-11.6`, `§ TODAY`, gap, informative):** `packages.FunctionMetadata`
declares `Concurrency`/`RateLimit` (`internal/packages/function_metadata.go:14-15`) and
`workflow_func.go:102-109` reads both, but neither `workflow.FunctionMetadata` (the wire type,
`pkg/workflow/metadata.go:6-10`) nor either DTO has a producer for them — both are always `nil` for
every registered package today, internal or remote. A remote capability that cannot declare a
concurrency cap will hammer the customer's endpoint the moment a workflow fans out; this document
does not add the fields (out of scope: not one of the backlog's named eight decisions) but flags it
in Appendix B as a gap B-02/B-03 should not ship without addressing.

**Cost to reverse (`INFORMATIVE`):** medium — the schema shape itself is public and additive once
published; the DTO/edges gap is a pure engine defect with no protocol-versioning cost to fix.

---

## 12. Decision 8 — Versioning

**Decision, protocol versioning (`NORMATIVE`, `RNP-12.1`):** the envelope carries `protocolVersion`
(`RNP-6.2`), a `"<major>.<minor>"` string (e.g. `"1.0"`). A worker declares, at capability
registration time, the set of protocol versions it supports (`supportedProtocolVersions:
["1.0"]`). FUSE dispatches using the **highest mutually supported major.minor** the engine itself
implements. If no overlap exists, registration MUST be rejected with a `400` naming the mismatch.
Minor versions within a major MUST be backward compatible (additive fields only, per `RNP-0.1`); a
major version bump MAY break wire shape and requires a worker to opt in explicitly (no implicit
upgrade).

**Decision, capability versioning (`NORMATIVE`, `RNP-12.2`):** a capability's version is embedded in
its registered id (e.g. `myco/billing@2`), not tracked as a separate mutable field on an otherwise
stable id. Registering a new version is registering a **new** package/function id; a graph node's
`functionId` reference is fixed at authoring time (already true today — `populateNodeMetadata`,
`internal/services/graph_service.go:295`, binds by splitting `node.Function` at the last `/` and
resolving `Registry.Get`/`GetFunctionMetadata` once) and does not silently follow a version bump.

**Engine constraint forcing this (`§ TODAY`, `RNP-12.3`):** there is no representation to build a
"same id, new version, live swap" model on top of even if this document wanted one:
`workflow.Package{ID, Functions, Tags}` (`pkg/workflow/package.go:14-16`) has no version field, and
the only versioning primitive anywhere in the engine is **graph-schema** versioning
(`internal/workflow/versioned_schema.go`), which is a different axis entirely (it versions the graph
that references capabilities, not the capabilities themselves) and is pinned per run by backlog
**A-05**.

**In-flight-run interaction with A-05 and B-08 (`RNP-12.4`, `NORMATIVE`):** because a graph node's
`functionId` is resolved once at bind time and capability version is embedded in that id
(`RNP-12.2`), an in-flight execution that already dispatched against `myco/billing@2` MUST continue
to resolve `myco/billing@2` for the remainder of that run, with no protocol-level renegotiation —
this falls out of `RNP-12.2` for free and does not require new engine machinery beyond what A-05
already needs for schema pinning. What this document explicitly does **not** provide: a way to
repoint the *same* capability id at a different physical worker instance without a version bump
(e.g. blue/green deploying a worker fleet behind one logical capability). That is a **worker
addressing / lifecycle** question — B-08's subject (`workflow.Package` has no multi-worker addressing
model today, per the `remote-node-protocol` skill) — and is deferred there, not solved by protocol
versioning.

**Rejected alternative:** a single mutable capability id whose schema/behaviour can change in place
(no version in the id), with FUSE tracking "current" vs. "as-of-run" versions itself. Rejected
because it requires exactly the kind of durable version-pinning infrastructure A-05 is already
building for graph schemas, and duplicating that machinery for capabilities is unjustified before
A-05 ships and is exercised — versioning-in-the-id is strictly cheaper and composes with A-05's
existing `(schemaID, version)` resolution (`FindByIDAndVersion`) without new engine work.

**Cost to reverse (`INFORMATIVE`):** high — every SDK's registration payload and every graph author's
node references negotiate against this from the first capability they register.

---

## 13. Worker registration — what this protocol version constrains (`INFORMATIVE`)

Full worker lifecycle (health checks, deregistration, draining, heartbeats, multi-worker addressing
behind one capability) is backlog **B-08** and is not designed here. This document constrains only
what a registration payload must be able to say for §6–§12 to be implementable: capability id
(`RNP-12.2`), supported protocol versions (`RNP-12.1`), declared input/output schema and edges
(`RNP-11.1`/`RNP-11.3`), the dispatch endpoint URL, and the auth secret(s) of `RNP-10.1`/`RNP-10.5`.
`internal/services/package_service.go:80`'s registration flow today is `Validate` → repository
`Save` → `Registry.Register`, a plain upsert with none of health-check, TTL, or drain semantics — a
`§ TODAY` fact this document leaves for B-08 to build on top of, not one it contradicts.

---

## Appendix A — Engine primitive inventory (personally confirmed by reading)

| Primitive | `file:symbol` | What it is |
| --- | --- | --- |
| `transport.Type` enum | `pkg/transport/type.go:5-11` | `HTTP Type = "http"` (exported, unread by production code); `gRPC Type = "grpc"` (lowercase — unexported, unreachable outside its own package). |
| `FunctionTransport` interface | `internal/packages/transport/function.go:10-18` | `Execute(actor.Handle, *ExecutionInfo)` (async-capable, binds `Finish`) + `ExecuteSync(*ExecutionInfo)` (no handle, for in-process tool calls). |
| `InternalFunctionTransport` | `internal/packages/transport/internal.go:34-79` | The only implementation. `Execute` binds `execInfo.Finish` to `sendAsyncFunctionResult`, which `gen.Node.Send`s to the workflow handler **by registered `gen.Atom` name**, not PID (`:12-24`, comment explains why PID `Send` from a pool worker fails). |
| Registration's executable/metadata-only branch | `internal/packages/loaded_package.go:112-128` | `if function.Metadata.Transport == transport.Internal && function.Function != nil` → executable; else metadata-only, `Transport: nil`. |
| "No transport" error | `internal/packages/loaded_package.go:41` (and `:56` for the sync path) | `"function %s has no transport (package likely loaded from persistence without its code-backed function)"`. |
| `Transport` hardcoded on registration | `internal/packages/loaded_package.go:70` | `MapToRegistryPackage` always writes `Transport: transport.Internal` into the registry struct, regardless of the incoming wire value. |
| `ExecutionInfo` | `pkg/workflow/execution_info.go:14-23` | Five fields: `WorkflowID`, `ExecID`, `Environment`, `Input`, `Finish`. No context, nodeId, threadId, attempt, or deadline. |
| `NewFunctionResultAsync` | `pkg/workflow/fn_result.go:44-49` | `FunctionResult{Async: true, Output: NewFunctionOutput(FunctionSuccess, nil)}`. |
| Async callback route | `internal/actors/mux_worker.go:79` | `POST /v1/workflows/{workflowID}/execs/{execID}`, 10s timeout, `PoolSize: 3`. |
| Async callback handler | `internal/handlers/async_function_result.go:53-85` | Reads path params, binds `dtos.AsyncFunctionRequest`, unconditionally `Send`s, returns `200`. No existence/pending/duplicate/format validation. |
| `dtos.AsyncFunctionRequest` | `internal/dtos/workflow.go:28-31` | `{Result workflow.FunctionOutput}` — one field, no validate tag. |
| `ExecuteFunctionMessage` | `internal/messaging/execute_function.go:12-20` | `WorkflowID, ExecID, ThreadID, PackageID, FunctionID, Input, Environment` — 7 fields; only 4 reach `ExecutionInfo`. |
| `Message.TraceCarrier` | `internal/messaging/message.go:44` | `map[string]string`, W3C trace-context headers — exists one level up from `ExecuteFunctionMessage`, not threaded into `ExecutionInfo`. |
| `ParseValue`'s type set | `internal/typeschema/parse.go:20-42` | `string, int, float64, bool, []byte, map[string]any`, plus `[]T` of those. Anything else: `"unsupported type: %s"`. |
| Coercion failures swallowed | `internal/workflow/workflow.go:652` (`inputMapping`), `:727` (`applyFlowMapping`) | Every failure branch logs and `continue`/`break`/`return`s; no error reaches the caller. |
| `HandleNodeFailure` | `internal/workflow/workflow.go:841` | One `RetryPolicy` applied uniformly; no taxonomy branch exists. |
| `findErrorEdges` | `internal/workflow/workflow.go:899` | Fires only after retries exhausted; follows only `errorEdges[0]`. |
| `RetryFunctionAction.Attempt` | `internal/workflow/workflowactions/action.go:55` | Exists, set in `HandleNodeFailure`, never threaded to `ExecuteFunctionMessage`/`ExecutionInfo`. |
| `ExecID.Thread()` | `pkg/workflow/exec_id.go:23-31` | Decodes thread id from UUID bytes 6-7; discards the `hex.DecodeString` error. |
| `RetryTracker` | `internal/workflow/retry_tracker.go` | In-memory attempt counter, not journalled (backlog A-08). |
| `SetResultFor` before the async check | `internal/actors/workflow_handler.go:275` vs. `:277` | Journals `step:completed` at dispatch time, before checking `Result.Async`. |
| `findPendingThreads` | `internal/workflow/workflow.go:267` | Matches `step:started` without completion — does not see an in-flight async call as pending, given the above. |
| Idempotency store | `internal/idempotency/store.go:7-19` | `Check/Set/Delete/CheckAndSet`, opaque string keys, trigger-scoped only (ADR-0017). |
| `InputMetadataDTO` / `OutputMetadataDTO` | `internal/dtos/package.go:42-45`, `:58-62` | No `Edges` field on either. |
| `MapToRegistryPackage`'s edge mapping | `internal/packages/loaded_package.go:74-77`, `:99-110` | Does map `Edges` from `workflow.Package` — the gap is upstream, at the DTO. |
| `FunctionMetadataDTO.Transport` swagger example | `internal/dtos/package.go:36` | `example:"sync"` — not a valid `transport.Type` value; cast through with no validation (`:178`). |
| Concurrency/RateLimit no producer | `internal/packages/function_metadata.go:14-15`, `pkg/workflow/metadata.go:6-10` | Consumer exists (`internal/actors/workflow_func.go:102-109`); no wire-type or DTO field ever sets them. |
| `pkg/secrets` | `pkg/secrets/credential.go:15`, `value.go:33` | `CredentialSecretName(id, field)` → `cred/<id>/<field>`; `SecretValue.Reveal()` is the only path to plaintext. |
| Auth greenfield | `docs/API.md:13`, `internal/handlers/handler.go:59` | "Not required today"; `Access-Control-Allow-Origin: *`; no middleware in `mux_server.go`. |
| `workflow.Package` | `pkg/workflow/package.go:14-16` | `{ID, Functions, Tags}` — no version field. |
| Graph-schema versioning (the only versioning that exists) | `internal/workflow/versioned_schema.go` | Different axis: versions the graph, not a capability. |
| `PUT /v1/packages/{packageID:.+}` ignores its path param | `internal/services/register_package.go:61` (used only for logging); saved id comes from the JSON body | Registration-path defect noted for completeness, not fixed here. |

## Appendix B — Engine gaps this spec creates (route to implementation, not fixed here)

| Clause | What the engine/worker must grow | Owning backlog id |
| --- | --- | --- |
| `RNP-6.5` `nodeId` on the dispatch path | Add `NodeID` to `ExecuteFunctionMessage` and `ExecutionInfo`; source it from the same audit-entry data that already produces `FunctionNodeID`. | B-02 |
| `RNP-6.6` `threadId` reaching the function | Add `ThreadID` to `ExecutionInfo` (already on `ExecuteFunctionMessage`, just not threaded further). | B-02 |
| `RNP-6.7`/`RNP-9.5` `attempt` in the envelope and idempotency key | Thread `RetryFunctionAction.Attempt` through `ExecuteFunctionMessage` into `ExecutionInfo`; journal it durably (backlog **A-08**). | A-08, then B-02 |
| `RNP-6.9` `deadline` in the envelope | Surface `TimeoutConfig.Execution`'s computed deadline into the envelope; note ADR-0023's caveat that the remote side keeps running after the engine's own deadline fires — a worker cannot assume cancellation. | B-02 |
| `RNP-6.13` `callbackUrl` construction | Add a server public-base-URL config (no such field exists; `LLMProviderConfig.BaseURL` is unrelated) and construct the absolute callback URL at dispatch time. | B-02 |
| `RNP-6.14`/`RNP-10.2`-`10.3` `callbackToken` minting, validation, single-use enforcement | New per-exec token issuance and a validation table keyed by `(workflowId, execId, attempt)`. | B-02 |
| `RNP-6.15` `traceparent` end-to-end | Blocked on backlog **A-04**: `_ = nodeCtx` (`internal/actors/workflow_func.go:96`) discards span context; `ExecutionInfo` has no `context.Context`/carrier field. ADR-0027 forbids re-adding a per-execution runtime *handle* as an `ExecutionInfo` field — a plain string/map carrier field is a distinct, narrower addition A-04 must settle. | A-04, then B-02 |
| `RNP-7.6`/`RNP-7.7` callback validation (existence/pending/duplicate/format) | Add the checks; template is `internal/handlers/resolve_awakeable.go:66-73`. | B-02 |
| `RNP-8.1`-`RNP-8.3` `errorType` taxonomy field | New wire field on the error payload; `HandleNodeFailure` must branch on it (skip retry loop for `business`, apply `RetryPolicy` for `infrastructure`). | B-02 |
| `RNP-9.2` idempotency key derivation | No producer exists for `(execId, attempt)`-derived keys anywhere; follow the shape of the existing deterministic `cron:`/`evt:` keys (ADR-0017). | C-10 |
| `RNP-10.1`-`10.5` auth (both directions) | 100% new: token issuance/storage via `pkg/secrets`, middleware in `internal/actors/mux_server.go` (none exists today), replay/clock-skew handling. | B-02 |
| `RNP-11.1` registration-time schema validation against `ParseValue`'s type set | No registration-time validator exists; `PUT /v1/packages/{packageID}` accepts any `type` string uncritically. | B-03 |
| `RNP-11.3`/`RNP-11.4` edge metadata on the REST DTO | Add `Edges` to `InputMetadataDTO` and `OutputMetadataDTO`; wire through `FromPackagedFunctionDTO`. | B-02 / B-03 |
| `RNP-11.6` concurrency/rate-limit producer | Add fields to `workflow.FunctionMetadata` and both DTOs; `MapToRegistryPackage` must set them. | B-02 / B-03 |
| `RNP-12.1` protocol-version negotiation at registration | No `protocolVersion` concept exists anywhere; add to the registration payload and dispatch-time resolution. | B-02 / B-08 |
| A-03's replay-vs-idempotency decision, formally recorded | This spec states the outcome (`RNP-9.1`); A-03 still owns writing and accepting its own ADR referencing this clause. | A-03 |

## Appendix C — Example wire messages (`INFORMATIVE`)

**Dispatch (FUSE → worker), synchronous success:**

```http
POST /fuse-worker/billing/charge HTTP/1.1
Authorization: Bearer wat_9f2c...
Content-Type: application/json

{
  "protocolVersion": "1.0",
  "workflowId": "wf_01J9Z8",
  "execId": "018f2e6a-...",
  "nodeId": "charge-card",
  "threadId": 0,
  "attempt": 0,
  "idempotencyKey": "018f2e6a-...:0",
  "deadline": "2026-09-10T12:00:30Z",
  "environment": "production",
  "functionId": "myco/billing@2/charge",
  "input": { "amount": 4200, "currency": "usd", "customerId": "cus_123" },
  "callbackUrl": "https://engine.example.com/v1/workflows/wf_01J9Z8/execs/018f2e6a-...",
  "callbackToken": "cbt_7a1e...",
  "traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
}
```

```http
HTTP/1.1 200 OK
Content-Type: application/json

{ "status": "success", "data": { "chargeId": "ch_789" } }
```

**Dispatch, async accepted then business-error callback:**

```http
HTTP/1.1 202 Accepted
Content-Type: application/json

{ "accepted": true }
```

```http
POST /v1/workflows/wf_01J9Z8/execs/018f2e6a-... HTTP/1.1
Authorization: Bearer cbt_7a1e...
Content-Type: application/json

{ "result": { "status": "error", "errorType": "business", "data": { "error": "invoice already voided" } } }
```

```http
HTTP/1.1 200 OK
Content-Type: application/json

{ "code": "OK", "workflowId": "wf_01J9Z8", "execId": "018f2e6a-..." }
```
