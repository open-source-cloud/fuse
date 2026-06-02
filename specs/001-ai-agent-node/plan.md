# Implementation Plan: AI Agent Node

**Branch**: `001-ai-agent-node` | **Date**: 2026-06-02 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-ai-agent-node/spec.md`

## Summary

Add an `ai/agent` workflow node (package `fuse/pkg/ai`) that reasons in a loop: it calls the LLM
provider with the configured task, lets the model request **tools**, invokes the corresponding
existing FUSE package functions **in-process**, feeds their results back to the model, and loops
until a final answer or a bounded iteration/time limit. The node completes **asynchronously**
(via `ExecutionInfo.Finish`, mirroring `ai/chat` and `logic/timer`) so it never occupies a
`WorkflowFunc` pool slot during the multi-step interaction. Tools are derived from the package
registry; Phase B exposes only **synchronous, declared-parameter** functions (ADR-0007).

## Technical Context

**Language/Version**: Go 1.26  
**Primary Dependencies**: ergo.services (actor model), uber-go/fx (DI), `openai-go/v3` (via the
existing `pkg/llm` provider abstraction), zerolog. No new third-party dependency.  
**Storage**: N/A — agent state is per-execution and in-memory; results flow through the existing
journal/aggregatedOutput like any node.  
**Testing**: `go test` / `gotestsum` (`make test`), table-driven + Arrange-Act-Assert, scripted
stub LLM provider + fake tool registry/handle (no live model needed).  
**Target Platform**: Linux server (the FUSE engine).  
**Project Type**: single project (Go module).  
**Performance Goals**: the agent must not reduce concurrent node-execution capacity for the
duration of its interaction (async completion); per-interaction time bound ~5 min.  
**Constraints**: must not introduce an import cycle (`ai` ↛ `internal/packages`); must keep
`pkg/workflow` free of `internal/*` type dependencies; exactly one `Finish` per run.  
**Scale/Scope**: one new node + one seam interface + one registry adapter + an `ExecutionInfo`
field + transport wiring. ~3 new files, ~5 modified, plus tests and one example workflow.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. Test-First (NON-NEGOTIABLE)**: PASS by construction — every task in `tasks.md` writes the
  test before (or alongside) the implementation; the scripted-provider harness exercises the loop
  without a live model. Core file `agent.go` targets ≥90% (critical business logic); the registry
  adapter targets ≥80% (repository/registry tier).
- **II. Code Quality Gates (NON-NEGOTIABLE)**: PASS — `make lint && make build && make test`
  after each phase; cyclomatic complexity ≤15 (the loop is factored into helpers to stay under).
- **III. Go Best Practices**: PASS — interface seam (`ToolRegistry`), wrapped errors, no shared
  mutable state across goroutines (the agent goroutine owns its own message slice).
- **IV. Actor Model Architecture**: PASS — reuses the `WorkflowFunc` worker + async `Finish`
  message path; the agent goroutine uses `handle.Node()` (never `Process.Send`) for any actor
  messaging, consistent with the documented ergo Sleep-state constraint.
- **V. DDD / VI. Clean+Hexagonal**: PASS — `ai` depends on a port (`ToolRegistry`) it owns; the
  adapter (infrastructure) lives in `packages` and is injected, preserving dependency inversion.
- **VII. Microservices / VIII. Concurrency**: PASS — no shared state; the goroutine is bounded by
  `context.WithTimeout`; the worker pool is freed immediately.

**No violations** → Complexity Tracking table omitted.

## Project Structure

### Documentation (this feature)

```text
specs/001-ai-agent-node/
├── plan.md              # This file
├── research.md          # Phase 0 output — design decisions & rationale
├── data-model.md        # Phase 1 output — entities (tool, tool call, step, conversation)
├── quickstart.md        # Phase 1 output — how to configure & run an agent node
├── contracts/
│   └── ai-agent-node.md # Node contract: inputs, outputs, behaviour (no new REST endpoint)
├── checklists/
│   └── requirements.md  # Spec quality checklist (from /speckit.specify)
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
pkg/workflow/
└── execution_info.go            # clean {WorkflowID, ExecID, Input, Finish} contract — no Handle field

internal/packages/
├── transport/function.go        # ExecuteSync(*ExecutionInfo) added to FunctionTransport
├── transport/internal.go        # InternalFunctionTransport.ExecuteSync (handle-free); Execute unchanged for async
├── loaded_package.go            # ExecuteFunctionSync(functionID, execInfo) — handle-free
├── agent_tools.go               # NEW: registry adapter implementing ai.ToolRegistry + exclusion predicate
├── internal_packages.go         # MODIFY: NewInternal(providers, registry) → build adapter → ai.New(providers, tools)
└── functions/ai/
    ├── tools.go                 # NEW: ToolRegistry/ToolDescriptor seam, mangle/demangle, schema conversion
    ├── agent.go                 # NEW: AgentFunctionMetadata + makeAgentFunction reasoning loop
    ├── package.go               # MODIFY: New(providers, tools) registers chat + agent
    ├── tools_test.go            # NEW
    ├── agent_test.go            # NEW
    └── chat.go                  # REUSE: resolveProvider / optionalTemperature helpers

internal/services/
└── graph_service*_test.go       # MODIFY: 9 NewInternal(...) call sites gain a registry arg

examples/workflows/
└── ai-agent-example.json        # NEW (CI-gated)

docs/adr/
├── 0007-...md                   # MODIFY: Proposed → Accepted
└── README.md                    # MODIFY: status bump
```

**Structure Decision**: Single Go module; the feature lives entirely in the existing
`internal/packages` layer (function registry + node functions), leaving `pkg/workflow`'s
`ExecutionInfo` contract unchanged. No new top-level packages; the only new files are the agent
node, its tool seam, and the registry adapter.

## Phase 0 — Research

See [research.md](research.md). Key decisions: (R1) tool invocation via a construction-injected
`ai.ToolRegistry` port over a **handle-free** `ExecuteSync` path (no `ExecutionInfo.Handle`); (R2)
the `ai.ToolRegistry` seam to break the import cycle; (R3) sync-only tool execution semantics; (R4)
the exclusion predicate; (R5) `ParameterSchema` → JSON-Schema mapping; (R6) name mangling; (R7)
loop termination & single-`Finish` discipline.

## Phase 1 — Design & Contracts

- [data-model.md](data-model.md) — the agent's in-memory entities and the node's input/output
  metadata.
- [contracts/ai-agent-node.md](contracts/ai-agent-node.md) — the node contract (no new HTTP
  endpoint; the agent is reachable through the existing `POST /v1/workflows/trigger`).
- [quickstart.md](quickstart.md) — configure and run an agent node end-to-end with Ollama.

## Complexity Tracking

No constitution violations — table intentionally omitted.
