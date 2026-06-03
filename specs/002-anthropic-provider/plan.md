# Implementation Plan: Native Anthropic LLM Provider

**Branch**: `002-anthropic-provider` | **Date**: 2026-06-02 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/002-anthropic-provider/spec.md`

## Summary

Add `internal/llm/providers/anthropic` — an `llm.Provider` implementation backed by
`anthropic-sdk-go` (v1.46.0) — and register it under the `anthropic` key. It mirrors the existing
`openaicompat` provider's shape (a `Config` + `Provider` + `New` + `Chat`), translating the
engine's provider-agnostic `llm.*` types to/from Anthropic's native Messages API (separate system
field, `tool_use`/`tool_result` content blocks, `input_schema` tools). This completes the Phase C
provider-breadth item in ADR-0006; the agent and chat nodes use it unchanged.

## Technical Context

**Language/Version**: Go 1.26  
**Primary Dependencies**: `github.com/anthropics/anthropic-sdk-go v1.46.0` (new direct dep), behind
the existing `pkg/llm` abstraction; uber-go/fx (DI), zerolog.  
**Storage**: N/A.  
**Testing**: `go test` / `gotestsum`; an `httptest` stub server pointed at via `option.WithBaseURL`
(same pattern as `openaicompat/provider_test.go`), returning Anthropic-shaped JSON — no live key.  
**Target Platform**: Linux server.  
**Project Type**: single Go module.  
**Performance Goals**: parity with existing providers; one network round-trip per `Chat`.  
**Constraints**: no node-side or registry-interface changes (`llm.Provider` is the seam); no new
config fields (the `anthropic` per-provider config already exists).  
**Scale/Scope**: one new package (~1 file + test), a DI edit, a `go.mod` dep. Streaming and prompt
caching are explicitly out of scope.

## Constitution Check

- **I. Test-First (NON-NEGOTIABLE)**: PASS — mapping is verified by table-driven tests against a
  stub endpoint before/with implementation; target ≥90% on the provider's mapping logic (critical).
- **II. Code Quality Gates (NON-NEGOTIABLE)**: PASS — `make lint && make build && make test`;
  complexity ≤15 (translation factored into small `to*/from*` helpers like `openaicompat`).
- **III. Go Best Practices**: PASS — implements the `llm.Provider` interface; wrapped errors with a
  `anthropic[name]:` prefix; SDK-agnostic types stay in `pkg/llm`.
- **IV–VIII**: PASS — no actor/DI/concurrency changes beyond registering one provider; the provider
  is stateless and safe for concurrent `Chat` calls (the SDK client is concurrency-safe).

**No violations.**

## Project Structure

```text
internal/llm/providers/anthropic/
├── provider.go        # NEW: Config, Provider, New, Chat + to/from mapping helpers
└── provider_test.go   # NEW: stub-server table tests (text, tool_use, system split, usage, no-model error)

internal/app/di/llm.go # MODIFY: register anthropic.New when enabled; remove the Phase C warning
go.mod / go.sum        # MODIFY: promote anthropic-sdk-go to a direct dependency (go mod tidy)

docs/adr/0006-...md     # MODIFY (minor): note the native Anthropic provider shipped
```

**Structure Decision**: New leaf package alongside `openaicompat`, selected by the DI registry.
No change to `pkg/llm`, the registry interface, the config struct (the `anthropic` provider config
already exists), or any node.

## Phase 0 — Research

See [research.md](research.md): the exact `anthropic-sdk-go` v1.46.0 API and the `llm.*` ↔ SDK
mapping (system split, message/content blocks, tools + `input_schema`, tool_choice, response
parsing, usage, required `max_tokens` default).

## Phase 1 — Design

No new entities or endpoints (the provider is reached through the existing
`POST /v1/workflows/trigger`); no `data-model.md`/`contracts/` needed. Enabling Claude is covered in
[quickstart.md](quickstart.md).

## Complexity Tracking

No constitution violations — table omitted.
