---
description: "Task list for AI Agent Node implementation"
---

# Tasks: AI Agent Node

**Input**: Design documents from `/specs/001-ai-agent-node/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: REQUIRED — the project constitution mandates Test-First (NON-NEGOTIABLE). Tests are
written before/with implementation and must fail first. Targets: ≥90% for `agent.go` (critical),
≥80% for the registry adapter (registry tier).

**Organization**: grouped by user story (US1 P1, US2 P2, US3 P3) after a foundational phase that
all stories depend on.

## Format: `[ID] [P?] [Story] Description`
- **[P]**: different file, no dependency on another unfinished task → parallelizable.

## Path Conventions
Single Go module; paths are repo-relative from `/home/gustavo/fuse/core`.

---

## Phase 1: Setup

- [ ] T001 Confirm working branch `001-ai-agent-node` and that `go test ./...` is green before
  changes (baseline). No new tooling required.

---

## Phase 2: Foundational (Blocking Prerequisites)

**⚠️ No user story can be implemented until this phase is complete** — the handle plumbing and the
tool seam/adapter are prerequisites for any tool call.

- [ ] T002 [P] Add `Handle any` field to `workflow.ExecutionInfo` in
  `pkg/workflow/execution_info.go`; add `pkg/workflow/execution_info_test.go` asserting default
  `nil` and settability. (Test-first.)
- [ ] T003 Populate the handle: in `internal/packages/transport/internal.go` `Execute`, set
  `execInfo.Handle = handle` after the nil check, before `t.fn(execInfo)`. Extend/add
  `internal/packages/transport/internal_test.go` asserting the function receives a non-nil
  `Handle`. (Depends on T002.)
- [ ] T004 [P] Create `internal/packages/functions/ai/tools.go`: `ToolRegistry`/`ToolDescriptor`
  port types, `toolHandle` assertion interface (`Node() gen.Node` + `Send`), `mangle`/`demangle`,
  `parameterSchemaToJSONSchema`. Add `internal/packages/functions/ai/tools_test.go` (table-driven:
  mangle/demangle round-trips; per-type JSON-Schema mapping; required list; empty-params object).
  (Test-first.)
- [ ] T005 Create `internal/packages/agent_tools.go` (package `packages`):
  `NewAgentToolRegistry(registry Registry) ai.ToolRegistry`, `ListTools`, `InvokeTool`,
  `isExposableTool`. Add `internal/packages/agent_tools_test.go` asserting `InvokeTool` returns a
  sync result inline and unknown id → error. (Depends on T004 for the `ai.ToolRegistry` type.)

**Checkpoint**: handle reaches functions; tools can be listed and invoked in-process.

---

## Phase 3: User Story 1 - Agent solves a task using tools (Priority: P1) 🎯 MVP

**Goal**: An agent node runs a reasoning loop, calls eligible tools in-process, and returns a
final answer asynchronously.

**Independent Test**: configure an agent told to "add 2 and 3" with `logic/sum` available; confirm
the tool is invoked and the final output contains the answer.

- [ ] T006 [US1] Create `internal/packages/functions/ai/agent.go`: `AgentFunctionID`,
  `AgentFunctionMetadata()` (inputs/outputs per data-model.md), `makeAgentFunction(providers, tools)`,
  `ErrAgentInputRequired`, `agentTimeout`. Implement the async loop (validate → resolve provider →
  assert handle → build tools + mangled map → `NewFunctionResultAsync()` → goroutine loop with
  `Chat`/`ToolChoice:"auto"`, tool invocation, tool-result messages, `Finish` exactly once). Reuse
  `resolveProvider`/`optionalTemperature` from `chat.go`. (Depends on T002–T005.)
- [ ] T007 [P] [US1] `internal/packages/functions/ai/agent_test.go` (mirror `chat_test.go`):
  scripted stub provider + fake `ToolRegistry` + fake `toolHandle`. Cases: happy path with one
  tool call (assert final `output`, `tool` message threaded with matching `ToolCallID`); direct
  answer (no tool calls); missing `input` → **sync** `FunctionError`; unknown provider → sync
  error; nil `Handle` → sync error.
- [ ] T008 [US1] Wire it: `internal/packages/functions/ai/package.go` →
  `New(providers llm.Registry, tools ToolRegistry)` registering `chat` + `agent`;
  `internal/packages/internal_packages.go` → `NewInternal(providers, registry)` builds the adapter
  and calls `ai.New(providers, adapter)`. Update the **9** `NewInternal(...)` call sites in
  `internal/services/graph_service{,_versioning,_versioning_integration}_test.go`. Confirm the fx
  graph builds (`make build`) — no `di.go` change expected.

**Checkpoint**: US1 fully functional — `make lint && make build && make test` green.

---

## Phase 4: User Story 2 - Bound & observe agent behaviour (Priority: P2)

**Goal**: predictable termination + observable trace + usage.

**Independent Test**: low iteration limit against a looping task terminates with a clear outcome
and a populated trace.

- [ ] T009 [US2] In `agent_test.go`: `maxIterations` exhaustion → terminal `FunctionError`
  (`"max iterations reached"`), no infinite loop; clamp out-of-range `maxIterations` to `[1,25]`.
  Add the timeout guard test (context deadline → terminal error) if feasible with the stub.
- [ ] T010 [US2] In `agent_test.go`: `steps` trace lists each tool call (tool, arguments,
  result/error); `usage` aggregated across multiple iterations. Adjust `agent.go` output assembly
  if needed so both fields are always present on success.

**Checkpoint**: US1 + US2 hold; every run terminates and is observable.

---

## Phase 5: User Story 3 - Control which tools an agent may use (Priority: P3)

**Goal**: optional allowlist + safe exclusion of incompatible functions + resilient tool errors.

**Independent Test**: an agent restricted to a subset only calls that subset; incompatible
functions are never offered.

- [ ] T011 [US3] Implement `allowedTools` intersection in `agent.go` (unset = all eligible). Test
  in `agent_test.go` that only allowlisted tools are offered to the model.
- [ ] T012 [US3] In `agent_test.go`: model requests an unknown/disallowed tool → error tool
  message fed back, loop continues then finishes; a tool returning `FunctionError` is surfaced as
  a tool message and the run is not aborted; an `Async:true` tool result is treated as
  "unsupported".
- [ ] T013 [US3] In `agent_tools_test.go`: comprehensive exclusion matrix over the built-in set —
  EXCLUDES `system/sleep|wait|subworkflow|foreach`, `fuse/pkg/logic/timer`, `fuse/pkg/logic/if`
  (CustomParameters), `fuse/pkg/ai/chat`, `fuse/pkg/ai/agent`; INCLUDES a synchronous schema'd
  function (e.g. `fuse/pkg/logic/sum`).

**Checkpoint**: all three stories independently functional.

---

## Phase 6: Polish & Cross-Cutting

- [ ] T014 [P] Add `examples/workflows/ai-agent-example.json` (trigger → `fuse/pkg/ai/agent` with a
  `logic/sum`-style tool), CI-gated per `make examples-ci` conventions.
- [ ] T015 [P] Promote ADR-0007 `Proposed → Accepted`; update "More Information" with real paths +
  PR; bump the status in `docs/adr/README.md`. Optionally run `spec-to-adr` on this spec/plan to
  confirm no new decision surfaced beyond ADR-0007.
- [ ] T016 Run the quality gate: `make lint && make build && make test`; resolve any
  complexity-≤15 lint findings by extracting loop helpers.
- [ ] T017 Run `speckit.analyze` for cross-artifact consistency (spec ↔ plan ↔ tasks).

---

## Dependencies & Execution Order
- **Setup (T001)** → **Foundational (T002–T005)** → **US1 (T006–T008)** → US2 (T009–T010) → US3
  (T011–T013) → **Polish (T014–T017)**.
- Within foundational: T003 depends on T002; T005 depends on T004; T002∥T004.
- US2 and US3 exercise behaviour mostly implemented in T006 (`agent.go`); they may require small
  refinements to `agent.go` but are otherwise additive tests.

## Parallel Opportunities
- T002 ∥ T004 (different files, no dep).
- T007 can be authored alongside T006 (same story, test-first).
- T014 ∥ T015 (docs/example, independent files).

## Notes
- Tests fail before implementation (constitution I).
- Commit after each phase/logical group with a conventional-commit message.
- The 5 forward-looking AI ADRs (orchestrator, async-tool channel, prompt/memory, cost/usage,
  structured output) are **out of scope for this feature spec** and handled as a separate
  `docs(adr)` workstream.
