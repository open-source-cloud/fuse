# 0008. Settings, environments & secrets management

- Status: Proposed — **decision DEFERRED**
- Date: 2026-06-01
- Deciders: FUSE maintainers

## Context and Problem Statement

Every secret in FUSE today is a process-level environment variable parsed once at
startup (`internal/app/config/config.go` via caarlos0/env): the database DSN, S3
keys, the cluster cookie, etcd credentials, and the new per-provider LLM
`APIKey`. The LLM provider registry is built from that config **once** at startup
(`internal/app/di/llm.go`), so provider keys and base URLs are fixed for the
process lifetime.

That model breaks down for agent workflows. Different workflows, tenants, or
environments need **different** credentials — e.g. one tenant's OpenAI key vs
another's, a per-customer API token for an HTTP tool, staging vs production
endpoints. Not everything can or should be an env var, and secrets must never be
stored inline in workflow schemas (which are persisted and versioned) or leak into
logs/traces/journals.

We are recording the problem and candidate directions now; **the decision is
deliberately deferred** to a focused follow-up.

## Current state (what exists / what's missing)

- Inputs to a node resolve at runtime from `SourceSchema` (static literal) or
  `SourceFlow` (a prior node's output) via `EdgeSchema.Input []InputMapping`
  (`internal/workflow/edge_schema.go`); functions receive a `*workflow.ExecutionInfo`
  (`pkg/workflow/execution_info.go`). **There is no secret-injection hook.**
- A `{{token}}` replacement utility exists (`pkg/strutil/strings.go` `ReplaceTokens`),
  used only by `debug/print` — a precedent for reference syntax, not a secrets system.
- **No** credentials/connections abstraction, **no** multi-tenancy/namespace concept,
  **no** secret references in schemas, **no** per-execution tenant/environment context
  is carried through a run.

## Decision Drivers / Requirements

- **Dynamic, context-scoped resolution**: pick the right secret per workflow /
  tenant / environment / execution — not a single process-wide value.
- **Central management**: add / update / rotate secrets without redeploying or
  restarting the engine.
- **Reference, never inline**: schemas reference a secret by name; plaintext secrets
  never appear in graph JSON, the journal, logs, or traces (redaction required).
- **Pluggable backends**: in-memory/dev, a persisted store, and external managers.
- **Least privilege & auditability**: a function receives only the resolved secrets
  it needs; access is auditable.
- Should compose with a future **multi-tenancy** model rather than fight it.

## Considered Options (no decision yet)

### Option A — Secret references + pluggable `SecretStore`, resolved at input mapping

Introduce a `{{secret:NAME}}` (or typed `SourceSecret` mapping) that a `SecretStore`
interface resolves during input mapping, scoped by the execution's context. Reuses
the existing `ReplaceTokens`/`InputMapping` precedent.

- Good: small, incremental; fits the current input pipeline; backend-agnostic.
- Bad: resolution logic spreads into the mapping layer; care needed so resolved
  values never get journaled; "environments" still needs a separate scoping concept.

### Option B — n8n-style credentials registry

First-class, typed **credential** objects (e.g. "OpenAI prod", "Acme webhook")
stored in a credentials repository and referenced by nodes/agents by ID; the engine
injects the resolved credential at execution.

- Good: familiar UX; reusable, typed, centrally managed; natural audit boundary.
- Bad: most new machinery (types, repository, CRUD API, UI); needs scoping/tenancy
  design; larger build.

### Option C — External secrets manager behind a `SecretStore` interface

Integrate HashiCorp Vault / Infisical / cloud KMS as a backend implementation of the
same `SecretStore` seam used by A or B.

- Good: enterprise-grade rotation, audit, encryption-at-rest; offloads custody.
- Bad: heavy external dependency; operational burden; likely a backend *for* A/B, not
  a standalone answer.

### Cross-cutting — an explicit "environments" concept

Independently of A/B/C, model **environments** (dev/staging/prod) as a scoping
dimension for both settings and secrets, and decide how/whether it couples to
multi-tenancy.

## Decision Outcome

**DEFERRED.** This ADR records the problem, current state, requirements, and the
option space. A follow-up ADR will choose a direction (the likely shape is a
`SecretStore` seam — Option A's reference syntax and/or Option B's credential
objects — with Option C as a pluggable backend), once the open questions below are
resolved. Until then, secrets remain env-var-based and process-wide.

### Open questions to resolve before deciding

- **Storage & encryption-at-rest**: which default backend; how secrets are encrypted
  in the persisted store; key management.
- **Scoping model**: workflow vs tenant vs environment vs user — and how this ties to
  a (not-yet-existing) multi-tenancy model.
- **Resolution point**: DI/startup vs input-mapping time vs a new `ExecutionInfo`
  hook (e.g. `ExecutionInfo.Secrets(name)`), and how dynamic per-execution selection
  reaches the LLM provider registry (which is currently static).
- **Rotation & caching**: TTLs, invalidation, behavior mid-execution.
- **Redaction**: guaranteeing resolved secrets never land in logs, traces, the
  journal, or `aggregatedOutput`.

## More Information

- Current config & wiring: `internal/app/config/config.go`, `internal/app/di/llm.go`.
- Runtime input path: `internal/workflow/edge_schema.go`,
  `pkg/workflow/execution_info.go`, `pkg/strutil/strings.go` (`ReplaceTokens`).
- Related: [ADR-0006](0006-llm-provider-abstraction-and-multi-provider-strategy.md)
  (static-at-startup registry this would make dynamic),
  [ADR-0005](0005-ai-agents-as-workflow-nodes-phased-roadmap.md) (agents are the main
  driver for per-context secrets).
