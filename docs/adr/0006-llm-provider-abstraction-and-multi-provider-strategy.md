# 0006. LLM provider abstraction & multi-provider strategy

- Status: Accepted
- Date: 2026-06-01
- Deciders: FUSE maintainers

## Context and Problem Statement

Agent and chat nodes ([ADR-0005](0005-ai-agents-as-workflow-nodes-phased-roadmap.md))
must talk to multiple LLM backends behind one interface: OpenAI, OpenRouter,
Ollama (local dev), Google Gemini, and Anthropic. We needed to choose how to
integrate these — adopt an agent framework, or build a thin abstraction over
official SDKs — and how to minimize per-provider code given that OpenAI,
OpenRouter, and Ollama are all OpenAI-wire-compatible (Gemini also exposes an
OpenAI-compatible endpoint), while Anthropic uses its own protocol.

## Decision Drivers

- One interface across all five providers; a single agent loop drives any of them.
- Fit FUSE's actor model and "own the control flow" philosophy.
- Minimal, stable dependencies; avoid Python-only paths.
- Local-dev friendly (Ollama, no API key).
- Keep the abstraction swappable so a framework could back it later.

## Considered Options

- **Google adk-go** (Agent Development Kit for Go).
- **Firebase Genkit Go** (GA; official multi-provider plugins).
- **CloudWeGo Eino** (graph-native agent framework).
- **Official SDKs behind a thin in-house `llm.Provider` interface** — `openai-go`
  for OpenAI-compatible backends, `anthropic-sdk-go` for native Claude; FUSE owns
  the agent loop.

## Decision Outcome

Chosen option: **official SDKs behind a thin `llm.Provider` interface.** FUSE
defines its own SDK-agnostic `pkg/llm` types (`Provider`, `Message`, `Tool`,
`ChatRequest`/`ChatResponse`). A single `openaicompat` implementation,
parameterized by base URL + API key + model, serves **OpenAI, OpenRouter, Ollama,
and Gemini's OpenAI-compatible endpoint** via `openai-go` — four providers from
one implementation. Native **Anthropic** gets its own implementation in Phase C.
A registry, built from configuration at startup, resolves providers by name.

This fits the actor model (we own the reasoning loop as our own code rather than
embedding a second orchestration framework), keeps dependencies minimal, and
keeps the interface clean enough that a Genkit- or Eino-backed `Provider` could be
substituted later without touching callers.

### Consequences

- Good: 4 of 5 providers from one SDK + base-URL switch; Ollama works with no key.
- Good: the agent loop is our code, integrating naturally with actors and async.
- Good: provider-agnostic types decouple us from any single SDK (swappable later).
- Bad: we maintain the provider mapping and the agent loop ourselves (no framework
  doing it for us).
- Bad: Anthropic's distinct protocol needs a separate mapping (deferred to Phase C).
- Neutral: the registry is built once at startup, so keys/base URLs are static per
  process — revisited by [ADR-0008](0008-settings-environments-and-secrets-management.md).

## Pros and Cons of the Options

### Official SDKs + thin interface (chosen)

- Good: minimal deps; full control; fits actors; one impl covers four providers.
- Bad: we own the loop and the per-provider mapping.

### Google adk-go

- Bad: in Go, multi-provider is effectively **Gemini-only** — its model-agnostic
  story depends on LiteLLM, which is Python-only. Wrong fit for a provider-neutral engine.
- Good: strong if committing to Gemini / A2A.

### Firebase Genkit Go

- Good: GA, Google-backed, official plugins for all five providers behind one interface.
- Bad: adds a second orchestration framework alongside ergo; heavier dependency. Kept
  as a viable fallback the `llm.Provider` seam could adopt later.

### CloudWeGo Eino

- Good: graph-native, excellent streaming, multi-provider via eino-ext.
- Bad: pre-1.0 (API churn); conceptually overlaps FUSE's own graph model.

## More Information

- Code: `pkg/llm/` (interface + registry), `internal/llm/providers/openaicompat/`,
  `internal/app/di/llm.go`; config in `internal/app/config/config.go` (`LLMConfig`).
- Shipped in PR #69 (Phase A).
- Native **Anthropic** provider (Phase C): `internal/llm/providers/anthropic/` via
  `anthropic-sdk-go`, registered under the `anthropic` key in `internal/app/di/llm.go`. Specified
  under `specs/002-anthropic-provider/`. (Streaming and prompt caching remain Phase C follow-ups.)
- Related: [ADR-0005](0005-ai-agents-as-workflow-nodes-phased-roadmap.md),
  [ADR-0007](0007-agent-reasoning-loop-and-tools-from-functions.md),
  [ADR-0008](0008-settings-environments-and-secrets-management.md).
