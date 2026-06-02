---
description: "Task list for the native Anthropic LLM provider"
---

# Tasks: Native Anthropic LLM Provider

**Input**: Design documents from `/specs/002-anthropic-provider/`
**Prerequisites**: plan.md, spec.md, research.md

**Tests**: REQUIRED (constitution: Test-First). Stub-server table tests, no live key.

## Phase 1: Setup

- [ ] T001 Add the dependency: `github.com/anthropics/anthropic-sdk-go@v1.46.0` (already fetched);
  `go mod tidy` will promote it to a direct dependency once it is imported (T002).

## Phase 2: Foundational — the provider (blocks the user stories)

- [ ] T002 Create `internal/llm/providers/anthropic/provider.go`: `Config{Name, APIKey, BaseURL,
  Model}`, `Provider{name, defaultModel, client}`, `New(cfg) *Provider` (build client via
  `option.WithAPIKey/WithBaseURL`), `Name()`, and `Chat(ctx, llm.ChatRequest) (llm.ChatResponse,
  error)` implementing the R1–R6 mapping (system split, message/tool blocks, tools + input_schema,
  tool_choice, response parse, usage, default `max_tokens` = 4096). Factor `to*/from*` helpers to
  keep complexity ≤15.
- [ ] T003 [P] Create `internal/llm/providers/anthropic/provider_test.go` (mirror
  `openaicompat/provider_test.go`): stub-server tests for (a) text response + usage + system/temp
  request shape, (b) tool advertisement + `tool_use` parse + `tool_choice`, (c) no-model error.
  (Test-first / alongside T002.)

## Phase 3: US1 + US2 — wire it in (the provider is usable by chat & agent)

- [ ] T004 [US1][US2] `internal/app/di/llm.go`: register `anthropic.New(...)` under the `anthropic`
  key when `cfg.LLM.Anthropic.Enabled`, log like the other providers, and **remove** the
  "not yet implemented (Phase C)" warning (FR-009). No registry/interface change.
- [ ] T005 [US1] `go mod tidy` so `anthropic-sdk-go` is a direct dependency; `go build ./...`
  confirms the fx graph still wires (no signature changes expected).

## Phase 4: US3 — configuration parity

- [ ] T006 [US3] Confirm the existing `LLM_ANTHROPIC_*` config (enable/api key/model/base url) flows
  through unchanged; add a base-URL field only if missing. Verify disabled ⇒ absent from registry,
  no warning (covered by reading config.go + a DI smoke check).

## Phase 5: Polish

- [ ] T007 [P] Minor ADR-0006 update: note the native Anthropic provider shipped (Phase C item).
- [ ] T008 Quality gate: `make lint && make build && make test`; then commit, push, open PR.

## Dependencies
- T002 → T003 (test targets the provider) and → T004 (DI imports the package).
- T004 → T005 (tidy/build) → T008.

## Notes
- The behavioural contract mirrors the shipped `openaicompat` provider; the agent/chat nodes and
  the `llm.Provider`/registry interfaces are unchanged.
- Out of scope: streaming, prompt caching (separate Phase C items).
