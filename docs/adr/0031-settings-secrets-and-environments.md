# 0031. Settings, secrets & environments: a SecretStore seam with secret references

- Status: Accepted
- Date: 2026-06-02
- Deciders: FUSE maintainers

## Context and Problem Statement

[ADR-0008](0008-settings-environments-and-secrets-management.md) recorded the problem — every
secret is a process-level env var parsed once at startup, the LLM provider registry is built once
([ADR-0006](0006-llm-provider-abstraction-and-multi-provider-strategy.md)) so keys/base-URLs are
fixed per process, there are no secret references in schemas, no per-context resolution, and no
redaction — and then **deferred the decision**. With the agent roadmap shipping
([ADR-0005](0005-ai-agents-as-workflow-nodes-phased-roadmap.md), 0007), the need is now concrete:
different workflows / environments / tenants need different credentials (e.g. one tenant's OpenAI
key vs another's, staging vs prod endpoints), and secrets must never be inlined in versioned
schemas or leak into logs/journal/traces. This ADR **resolves 0008** by choosing an architecture
and a phased path; it supersedes 0008. Implementation is phased and tracked separately (no code in
this ADR).

## Decision Drivers

- **Dynamic, context-scoped resolution** — the right secret per workflow / environment / (future)
  tenant / execution, not a single process-wide value.
- **Reference, never inline** — schemas reference a secret by name; plaintext never appears in
  graph JSON, the journal ([ADR-0010](0010-durable-execution-journal-and-replay.md)),
  `aggregatedOutput`, logs, or traces ([ADR-0020](0020-observability-metrics-tracing-execution-traces.md)).
- **Central management** — add / update / rotate without redeploying or restarting the engine.
- **Pluggable backends** — in-memory/dev, a persisted (encrypted-at-rest) store, and external
  managers, behind one seam.
- **Least privilege & auditability** — a function receives only the resolved values it needs.
- **Incremental & low-blast-radius** — fit the existing input pipeline; keep functions oblivious to
  secrets; compose with a future multi-tenancy model rather than fight it.

## Considered Options

(Carried forward from ADR-0008.)

- **A — Secret references + pluggable `SecretStore`, resolved at input-mapping time.** A typed
  `SourceSecret` mapping (and/or a `{{secret:NAME}}` reference) that a `SecretStore` resolves while
  building a node's input, scoped by the execution's context.
- **B — n8n-style credentials registry.** First-class, typed credential objects ("OpenAI prod")
  in a repository, referenced by id, injected at execution.
- **C — External secrets manager** (Vault / Infisical / cloud KMS) behind a `SecretStore` interface.
- **Cross-cutting — an explicit "environments" concept** (dev/staging/prod) as a scoping dimension.

## Decision Outcome

Chosen: **a layered architecture with Option A as the foundation**, because it satisfies the
drivers with the smallest, most incremental change and gives B and C a stable seam to build on.

1. **Foundation — `SecretStore` seam + typed secret references (A).** Introduce a `SecretStore`
   port (`Resolve(ctx, scope, name) (SecretValue, error)`) and a new **`SourceSecret`** input
   source (plus a `{{secret:NAME}}` reference form reusing the `ReplaceTokens` precedent in
   `pkg/strutil`). The **engine resolves references at input-mapping time**
   (`internal/workflow/edge_schema.go` / the input-mapping step), so a function receives
   already-resolved values and **never sees the store**. This deliberately mirrors the principle
   established by the ai/agent refactor ([ADR-0007](0007-agent-reasoning-loop-and-tools-from-functions.md)):
   runtime capabilities are injected at the engine layer, **not** hung on the `ExecutionInfo`
   input/output contract.
2. **Credential objects (B) layer on the same seam, later.** Typed, centrally-managed credentials
   referenced by id are a higher-level convenience implemented *over* `SecretStore` (a credential
   resolves to one or more secrets); they are not a competing design.
3. **External managers (C) are pluggable `SecretStore` backends.** Vault/Infisical/KMS implement
   the same interface; they are custody/rotation backends *for* A/B, not a standalone answer.
4. **Scoping = environment + workflow now, tenant-ready.** A resolution **scope** (environment
   label + workflow id, extensible to tenant) is passed *into* `SecretStore.Resolve` — carried by
   the engine, not stored on the function contract. An explicit **environment** concept
   (dev/staging/prod) is the first scoping dimension.
5. **Redaction is mandatory and part of the seam.** Resolved secrets are carried as a marked
   `SecretValue` type so they render as `[REDACTED]` and are stripped before anything reaches
   logs, the journal, traces, snapshots, or `aggregatedOutput`. A reference that is never resolved
   never materializes a plaintext value anywhere persisted.
6. **Provider keys (the agent driver) resolve via the same seam.** The static startup registry
   ([ADR-0006](0006-llm-provider-abstraction-and-multi-provider-strategy.md)) remains the default;
   per-context LLM keys/base-URLs resolve from `SecretStore`, which requires decoupling provider
   construction from startup DI — recorded as a later phase, not done now.
7. **Rotation & caching.** The store may cache with a TTL; resolution is per-execution so rotation
   takes effect on the next run; values are stable within a single execution.

### Phased roadmap

- **Phase 1** — `SecretStore` interface + in-memory/dev backend; `SourceSecret` / `{{secret:NAME}}`
  resolved at input mapping; the `SecretValue` redaction type wired through logging/journal/snapshots.
- **Phase 2** — credential objects (B): typed credentials repository + CRUD API/UI, referenced by id.
- **Phase 3** — external backends (C: Vault/Infisical/KMS), per-context LLM provider keys (dynamic
  provider construction), and the explicit environments model.

### Consequences

- Good: a single seam unifies dev/persisted/external secrets; references keep plaintext out of
  schemas; functions stay oblivious; the change is incremental and fits the existing input pipeline.
- Good: composes cleanly with future multi-tenancy (scope already threads through `Resolve`).
- Bad: resolution and redaction logic live in the engine's input/serialization layers and must be
  audited carefully so no plaintext path is missed; redaction has to cover every sink.
- Bad: per-context provider keys require decoupling provider construction from startup DI (Phase 3).
- Neutral: a chosen direction, not a full implementation — Phases 2–3 remain to be scheduled.

## Pros and Cons of the Options

### A — SecretStore seam + references (chosen foundation)
- Good: minimal, incremental, backend-agnostic; centralizes resolution + redaction at one point.
- Bad: care needed that resolved values never get journaled/logged; "environments" still needs its
  own scoping concept (addressed by the scope passed to `Resolve`).

### B — Credentials registry (chosen as a later layer)
- Good: familiar UX; reusable, typed, centrally managed; a natural audit boundary.
- Bad: most new machinery (types, repo, CRUD, UI) — justified only once the seam (A) exists.

### C — External secrets manager (chosen as a pluggable backend)
- Good: enterprise rotation/audit/encryption-at-rest; offloads custody.
- Bad: heavy dependency + operational burden; a backend *for* A/B, not standalone.

## More Information

- **Phase 1 shipped**: the `SecretStore` seam + `{{secret:NAME}}`/`source:"secret"` references
  (resolved in `internal/workflow/workflow.go` `inputMapping`), AES-256-GCM redaction via
  `pkg/secrets` (`SecretValue` → `***` everywhere; `FunctionInput.GetStr` reveals plaintext), the
  **memory** + **encrypted-Postgres** backends (`pkg/secrets/memory.go`,
  `internal/repositories/postgres/secret.go` + migration `000008_create_secrets`), the
  `SECRETS_DRIVER` selection (`internal/app/di/secrets.go`), and the `fuse secrets` CLI.
- **Phase 3 — explicit environments model shipped**: `environment` is now a per-execution
  scoping dimension. It is chosen at trigger time (`environment` on `TriggerWorkflowRequest`,
  `fuse workflow -e`), threaded through the trigger → supervisor → `WorkflowHandler` chain, and
  persisted on the `workflows` row (migration `000009`) so journal replay / recovery resolve
  secrets against the same environment; the engine builds a per-workflow `secrets.Resolver`
  scoped to it (replacing the process-wide one) and sub-workflows inherit the parent's
  environment. A first-class environments registry (`Environment` domain, memory +
  Postgres `EnvironmentRepository` + migration `000010` seeding `default`, `EnvironmentService`,
  CRUD at `/v1/environments`) makes environments declarable, and triggers naming an unknown
  environment are rejected (HTTP 400).
- **Phase 3 — per-context LLM provider keys shipped**: a provider's API key / base URL
  (`LLM_<PROVIDER>_API_KEY`, `LLM_<PROVIDER>_BASE_URL`) may be a `{{secret:NAME}}` reference
  resolved from the `SecretStore` against the running workflow's `environment`. `provideLLMRegistry`
  now builds per-provider `llm.ProviderFactory` closures instead of fixed providers
  (`internal/app/di/llm.go`): fully-static config is built once (fast path), reference-bearing
  config resolves and constructs a provider per execution. The `environment` reaches the
  ai/chat & ai/agent functions via `ExecutionInfo.Environment` (carried on
  `messaging.ExecuteFunctionMessage`); resolution runs in the function's async goroutine and the
  resolved key is `Reveal()`'d only into the SDK client, never logged. Different workflows in one
  process can now use different provider credentials per environment.
- **Phase 2 — credential objects shipped**: typed, centrally-managed credentials referenced by id.
  A `Credential` is metadata (`pkg/workflow/credential.go`: id, free-form type, field names) in a
  `CredentialRepository` (memory + Postgres, migration `000011_create_credentials`); its field
  **values** live in the `SecretStore` under the reserved `cred/<id>/<field>` name, per environment
  — one custody point, reusing Phase-1 encryption (the `/` separator is outside the `{{secret:}}`
  charset, so the namespaces cannot collide). `CredentialService` writes values via a
  `ManagedSecretStore`; CRUD at `/v1/credentials` (values never returned on reads) and a
  `fuse credentials` CLI manage them. Three reference forms — input-mapping `source:"credential"`,
  the `{{credential:id.field}}` token (`pkg/secrets/credential.go`), and
  `LLM_<PROVIDER>_CREDENTIAL=<id>` — all resolve through the existing environment-scoped resolver.
  Only **Infisical** (the read-only external backend, option C) remains scheduled.
- Supersedes [ADR-0008](0008-settings-environments-and-secrets-management.md) (which recorded the
  problem and deferred the decision).
- Current state this changes: `internal/app/config/config.go` (env-var secrets),
  `internal/app/di/llm.go` (static-at-startup provider registry), `internal/workflow/edge_schema.go`
  (`SourceSchema`/`SourceFlow` input mapping — gains `SourceSecret` and `SourceCredential`), `pkg/strutil/strings.go`
  (`ReplaceTokens` precedent), `pkg/workflow/execution_info.go` (stays a clean contract — no secret
  field).
- Related: [ADR-0005](0005-ai-agents-as-workflow-nodes-phased-roadmap.md) (agents drive per-context
  secrets), [ADR-0006](0006-llm-provider-abstraction-and-multi-provider-strategy.md) (the static
  registry this makes dynamic in Phase 3), [ADR-0013](0013-workflow-schema-model-and-input-mapping.md)
  (the input-mapping model `SourceSecret` extends), [ADR-0010](0010-durable-execution-journal-and-replay.md)
  / [ADR-0020](0020-observability-metrics-tracing-execution-traces.md) (sinks redaction must cover).
