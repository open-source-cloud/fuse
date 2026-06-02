# 0025. Browser automation & web-scraping package

- Status: Proposed
- Date: 2026-06-01
- Deciders: FUSE maintainers

## Context and Problem Statement

We want FUSE users to author **web-scraping automations as workflows**: navigate to a
page, fill forms, click, wait for elements, paginate, and extract structured data,
then flow that data on to downstream nodes. The package/function plumbing itself is
easy — it copies the `ai/chat` pattern ([ADR-0005](0005-ai-agents-as-workflow-nodes-phased-roadmap.md))
almost verbatim. The hard, costly-to-reverse questions are unique to browsers:
**how does a live browser session map onto the workflow graph**, and **how do we
integrate a heavy, stateful, non-serializable engine** without breaking the async,
journaled, multi-node execution model? A browser session lives in one process's
memory and cannot be serialized into the journal ([ADR-0010](0010-durable-execution-journal-and-replay.md))
or follow a workflow that is reclaimed onto another node after failover
([ADR-0018](0018-high-availability-and-clustering.md)).

## Decision Drivers

- Fit the existing package/function registry ([ADR-0024](0024-package-registry-and-function-metadata.md))
  and **always-async** execution model (`transport.Internal` + `execInfo.Finish()`);
  add no new executor concepts.
- **Survive failover** ([ADR-0018](0018-high-availability-and-clustering.md)): a
  reclaimed workflow may resume on a different node, but a live session cannot move
  with it. The session model must not assume a session outlives a single node's
  custody unless we explicitly add affinity.
- **Reuse existing control flow** — `logic/if`, `system/foreach`, `system/subworkflow`
  ([ADR-0013](0013-workflow-schema-model-and-input-mapping.md),
  [ADR-0015](0015-conditional-routing-with-expr-lang.md)) — instead of inventing
  browser-specific looping/branching.
- **Keep the engine swappable**, following the LLM provider precedent
  ([ADR-0006](0006-llm-provider-abstraction-and-multi-provider-strategy.md)); many
  "scrapes" need no real browser and a cheap HTTP-fetch engine should fit the same seam.
- **Resource safety**: a Chromium process is ~100–300 MB and ~1 s to launch — it must
  be a pooled DI singleton with back-pressure, never created per function call (unlike
  the per-request HTTP client in `internal/packages/functions/http/request.go`).
- Local-dev and Docker friendliness (the browser binary and its system libraries are a
  real packaging cost, tied to [ADR-0021](0021-deployment-and-delivery-architecture.md)).

## Considered Options

- **Engine integration:** (a) a thin `pkg/browser.Engine` seam with pluggable impls,
  vs (b) bind functions directly to `playwright-go`.
- **Session-to-node mapping:** (a) **coarse-only** — one node always runs one whole
  session; (b) **fine-grained-primary** — a `sessionId` threaded across many small
  nodes from day one; (c) **coarse default, fine-grained later (phased)**.

## Decision Outcome

Chosen options: **a `pkg/browser.Engine` provider seam (Playwright first), and a
phased session model — coarse `browser/run` node first, fine-grained nodes later.**

**Engine seam.** Define SDK-agnostic `pkg/browser` types (`Engine`, `Session`,
`SessionOptions`, step/result types) mirroring `pkg/llm`. The first implementation
wraps `github.com/playwright-community/playwright-go`; a lighter HTTP-fetch/goquery
engine and others (chromedp, a remote browser grid) can plug in behind the same
interface for scrapes that don't need a real browser. The engine and a **bounded
session pool** are DI singletons created once with `fx.Lifecycle` cleanup — the same
pattern as the pgx pool (`internal/app/di/database.go`) and `LLMModule`
(`internal/app/di/llm.go`) — never per request.

**Phased session model.**

- **Phase 1 — `browser/run` (coarse).** One node owns one entire session:
  launch → ordered steps (goto/fill/click/waitFor/extract) → close, all inside a
  single async function invocation. The session **never crosses a node boundary**, so
  it is HA-safe with **zero affinity work** and never touches the journal. The
  function returns `NewFunctionResultAsync()` and reports via `execInfo.Finish()`, and
  sets `Concurrency`/`RateLimit` metadata so the existing `WorkflowFunc` actor
  (`internal/actors/workflow_func.go`) back-pressures against the pool for free.
  In-page control flow is expressed as a step list parameter; cross-page orchestration
  (loop over results, branch on page state, reuse a login) uses the existing
  `system/foreach`, `logic/if`, and `system/subworkflow` nodes.
- **Phase 2 — fine-grained nodes (deferred).** `browser/launch` emits a `sessionId`
  threaded across `goto`/`fill`/`click`/`extract`/`close` nodes via edge input mapping
  (`source:"flow"`, [ADR-0013](0013-workflow-schema-model-and-input-mapping.md)), so the
  graph reads like a script. This requires **session affinity** — sticky routing of
  every node carrying a given `sessionId` back to the node that owns the browser — and
  the live session is lost on failover. It is deferred behind Phase 1 because affinity
  is the genuinely new mechanism, not the node code.

Coarse-first is chosen because it is the **only** model that is HA-safe with no new
routing machinery, ships fast, and reuses the engine's existing control flow — while
the seam keeps the more ergonomic fine-grained authoring open as a clean Phase 2 on
the same foundations.

### Consequences

- Good: Phase 1 is HA-safe by construction (session never leaves its node) and needs
  no affinity, journal, or executor changes.
- Good: reuses `foreach`/`if`/`subworkflow` for orchestration; no browser-specific
  control-flow primitives.
- Good: the `Engine` seam decouples FUSE from Playwright and lets a cheap HTTP-fetch
  engine serve no-JS scrapes through the same nodes.
- Good: pooled singleton + concurrency/rate-limit metadata gives back-pressure against
  the browser pool for free ([ADR-0016](0016-concurrency-and-rate-limiting.md)).
- Bad: until Phase 2, in-page control flow lives in a node's step-list parameter rather
  than as visible graph nodes.
- Bad: Phase 2 requires real **session-affinity routing** and remains fragile under
  failover — a deliberate, deferred cost.
- Bad: the **Docker image grows** — the browser binary plus ~20 system libraries
  (`playwright install --with-deps chromium`) materially enlarge the `fuse-app` image
  and CI; this must be budgeted explicitly ([ADR-0021](0021-deployment-and-delivery-architecture.md)).
- Neutral: screenshots and large extracts should ride the object store
  ([ADR-0019](0019-object-store-payload-externalization.md)) rather than inline payloads.
- Neutral: per-node concurrency limits are per-node only in HA
  ([ADR-0016](0016-concurrency-and-rate-limiting.md)) — pool sizing is a per-process concern.

## Pros and Cons of the Options

### Engine seam + Playwright first (chosen)

- Good: swappable engines (Playwright, HTTP-fetch, grid) behind one interface; matches
  ADR-0006; one node set serves multiple engines.
- Bad: a small amount of mapping code (FUSE types ↔ Playwright) we own and maintain.

### Bind directly to `playwright-go`

- Good: least code up front.
- Bad: couples every function to Playwright; swapping or adding a lighter engine later
  means rewriting each function; diverges from the established provider pattern.

### Coarse-only session model

- Good: simplest, fully HA-safe forever, no affinity ever.
- Bad: permanently loses the graph-as-script authoring experience; all in-page logic
  is buried in node parameters.

### Fine-grained-primary session model

- Good: best authoring UX — each browser action is a visible node, branch/loop with
  native `if`/`foreach`.
- Bad: needs session-affinity routing from day one and is fragile under failover; a
  much larger and riskier first step.

### Coarse default, fine-grained later (chosen)

- Good: usable and HA-safe now, ergonomic later, on shared engine/pool foundations.
- Bad: requires sequencing discipline and a future affinity decision (its own ADR).

## More Information

- Planned code: `pkg/browser/` (`Engine`/`Session` interfaces + types),
  `internal/browser/playwright/` (first engine impl), later
  `internal/browser/httpfetch/`; `internal/packages/functions/browser/` (`browser/run`
  in Phase 1; `launch`/`goto`/`fill`/`click`/`extract`/`close` in Phase 2);
  `internal/app/di/browser.go` (pooled singleton + `fx.Lifecycle` cleanup);
  `BrowserConfig` in `internal/app/config/config.go` (engine selection, pool size,
  headless, timeouts; disabled by default like the LLM providers).
- Pattern reuse: `internal/packages/functions/ai/chat.go` (async + `Finish`),
  `internal/packages/internal_packages.go` (package registration),
  `internal/app/di/llm.go` and `internal/app/di/database.go` (singleton lifecycle).
- Related: [ADR-0005](0005-ai-agents-as-workflow-nodes-phased-roadmap.md) (phased-roadmap
  precedent), [ADR-0006](0006-llm-provider-abstraction-and-multi-provider-strategy.md)
  (provider-seam precedent), [ADR-0013](0013-workflow-schema-model-and-input-mapping.md)
  (edge input mapping for `sessionId` threading),
  [ADR-0016](0016-concurrency-and-rate-limiting.md) (pool back-pressure),
  [ADR-0018](0018-high-availability-and-clustering.md) (failover constraint),
  [ADR-0019](0019-object-store-payload-externalization.md) (screenshot/large-extract
  payloads), [ADR-0021](0021-deployment-and-delivery-architecture.md) (image size),
  [ADR-0024](0024-package-registry-and-function-metadata.md) (the package/function
  contract a `browser` package plugs into).
- Follow-up decision this implies: a future ADR on **session-affinity routing** when
  Phase 2 (fine-grained nodes) is undertaken.
