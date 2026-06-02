# Feature Specification: Native Anthropic LLM Provider

**Feature Branch**: `002-anthropic-provider`  
**Created**: 2026-06-02  
**Status**: Draft  
**Input**: User description: "Native Anthropic provider: implement the llm.Provider interface against Anthropic's native Messages API via anthropic-sdk-go, so agent and chat nodes can use Claude models. Completes the Phase C provider-breadth item from ADR-0006."

## Clarifications

### Session 2026-06-02

- **Q: Default max output tokens when a request omits it?** → A: **4096** (Anthropic requires an
  explicit `max_tokens`; the engine supplies this default unless the node sets one).
- **Q: Default endpoint?** → A: the SDK default (`api.anthropic.com`), overridable via a base-URL
  setting for proxies/gateways and tests.
- **Q: Streaming / prompt caching in scope?** → A: **No** — separate Phase C items
  ([ADR-0006](../../docs/adr/0006-llm-provider-abstraction-and-multi-provider-strategy.md)).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Use Claude from a chat node (Priority: P1)

A workflow author who has an Anthropic API key configures the `anthropic` provider and points an
`ai/chat` node at it (or makes it the default), and the node returns a Claude completion — through
the exact same node configuration as any other provider.

**Why this priority**: This is the core deliverable — Claude is a top-tier model family and the
only one ADR-0006 left unimplemented. Without it, the provider abstraction's "any of five
providers behind one interface" promise is incomplete.

**Independent Test**: Configure `anthropic` (key + model), run a workflow whose chat node targets
it, and confirm a text answer comes back with token usage populated.

**Acceptance Scenarios**:

1. **Given** the `anthropic` provider is enabled with a key and model, **When** a chat node runs
   against it, **Then** it returns the model's text answer and a token-usage summary.
2. **Given** a node selects `provider: anthropic` while another provider is the default, **When**
   it runs, **Then** the Anthropic provider is used.
3. **Given** the provider call fails (bad key, model error), **When** the node runs, **Then** the
   failure surfaces as a node error through normal error handling, not a crash.

---

### User Story 2 - Use Claude from an agent node with tools (Priority: P2)

An author points an `ai/agent` node at `anthropic` and the agent's tool-calling reasoning loop
works unchanged: Claude is offered the same tools, requests tool calls, receives tool results, and
produces a final answer.

**Why this priority**: Tool-calling parity is what makes the new provider useful for the agent
(the headline Phase B feature), not just plain chat. It builds directly on P1.

**Independent Test**: Run the existing agent example workflow with `provider: anthropic` against a
stubbed/real endpoint and confirm a tool is invoked and a final answer returned — identical
behaviour to an OpenAI-compatible provider.

**Acceptance Scenarios**:

1. **Given** an agent node using `anthropic` with at least one eligible tool, **When** it runs,
   **Then** Claude can request a tool, the tool result is fed back, and a final answer is produced.
2. **Given** the same workflow run on `anthropic` vs an OpenAI-compatible provider, **When** both
   run, **Then** the agent's observable behaviour (tool calls, final output, usage) is equivalent.

---

### User Story 3 - Configure Anthropic like any other provider (Priority: P3)

An operator enables, disables, and configures the Anthropic provider through the same per-provider
configuration mechanism (enable flag, API key, model, optional base URL) as the existing
providers, with predictable behaviour when it is not configured.

**Why this priority**: Operational consistency; a refinement that makes the provider safe to ship
with sensible defaults.

**Independent Test**: Toggle the enable flag and confirm the provider appears/disappears from the
registry; with it disabled, nodes targeting it get a clear "unknown provider" error.

**Acceptance Scenarios**:

1. **Given** the Anthropic enable flag is off, **When** the engine starts, **Then** the provider
   is not registered and no "not implemented" warning is emitted.
2. **Given** it is enabled, **When** the engine starts, **Then** it is registered under the
   `anthropic` key and logged like the other providers.

### Edge Cases

- **No API key**: a clear configuration/auth error rather than a silent failure.
- **No model**: if neither the node nor the provider config specifies a model, the provider fails
  fast with a clear error (consistent with the OpenAI-compatible provider).
- **No max tokens**: Anthropic requires `max_tokens`; the provider supplies a default so requests
  that omit it still succeed.
- **Multiple system messages / tool results**: translated correctly into Anthropic's separate
  system field and tool_result blocks.
- **Provider/network error**: surfaced as a node error, never a crash.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST provide an `anthropic` LLM provider, usable by `ai/chat` and
  `ai/agent` through the same provider interface as the existing providers (no node-side changes).
- **FR-002**: The provider MUST support Claude **tool calling** — advertise tools, receive
  tool-use requests, and accept tool results — so the agent reasoning loop works unchanged.
- **FR-003**: The provider MUST translate the engine's provider-agnostic messages (system, user,
  assistant, tool) to and from Anthropic's native protocol, including its **separate system
  field** and `tool_use` / `tool_result` content blocks.
- **FR-004**: The provider MUST report token usage (input, output, and total) from Anthropic
  responses.
- **FR-005**: The provider MUST be enabled/disabled and configured (API key, model, optional base
  URL) via the same per-provider configuration mechanism as the other providers, and be selectable
  as the default provider.
- **FR-006**: When enabled but a required setting is missing (no model and no per-request model),
  the provider MUST fail with a clear error.
- **FR-007**: Because Anthropic requires an explicit maximum output-token count, the provider MUST
  supply a sensible default when a request omits it.
- **FR-008**: Provider, model, and network errors MUST surface as errors to the calling node
  (through normal error handling), never crashing the engine.
- **FR-009**: When the Anthropic provider is disabled, the engine MUST NOT emit a
  "not yet implemented" warning (that placeholder is removed once it is implemented).

### Key Entities *(include if feature involves data)*

- **Anthropic Provider**: a registered LLM provider (registry key `anthropic`) configured with an
  API key, default model, and optional base URL; satisfies the same interface as other providers.
- **Provider-agnostic message / tool / usage**: the existing engine types, mapped to/from
  Anthropic's native request/response shapes.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A chat node configured with `provider: anthropic` returns a Claude completion
  end-to-end with no changes beyond configuration.
- **SC-002**: An agent node using `anthropic` completes a tool-using task with the same observable
  behaviour as the same workflow on an OpenAI-compatible provider.
- **SC-003**: Token usage (input/output/total) is reported for Anthropic runs.
- **SC-004**: Switching a workflow's provider between `anthropic` and another requires only
  configuration — no workflow or code changes.
- **SC-005**: Enabling/disabling the provider is a pure configuration change; disabled means
  absent from the registry with no spurious warning.

## Assumptions

- **Default max output tokens**: 4096 when unspecified.
- **Default endpoint**: the SDK default (`api.anthropic.com`); overridable via a base-URL setting.
- **Out of scope**: streaming and prompt caching (separate Phase C items); these are not delivered
  here.
- **Configuration shape**: reuses the existing per-provider config (`enable`, `api key`, `model`,
  `base url`) already present for `anthropic` in the config struct.
