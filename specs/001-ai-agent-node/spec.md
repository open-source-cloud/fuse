# Feature Specification: AI Agent Node

**Feature Branch**: `001-ai-agent-node`  
**Created**: 2026-06-02  
**Status**: Draft  
**Input**: User description: "ai/agent node: an AI agent workflow node that reasons in a loop and calls existing FUSE package functions as tools, completing asynchronously like ai/chat. Grounded in ADR-0007 (Phase B of the AI roadmap). Phase B exposes only synchronous, schema'd package functions as tools (excludes async/intercepted functions like system/sleep, system/wait, system/subworkflow, system/foreach, logic/timer, and schemaless CustomParameters functions like logic/if). The agent invokes tools in-process, feeds results back as tool messages, and loops until a final answer or maxIterations."

## Clarifications

### Session 2026-06-02

The following bounded defaults were presented in, and approved as part of, the implementation
plan for this feature; they are recorded here to resolve the deferred items in **Assumptions**:

- **Q: Default and maximum number of reasoning iterations?** → A: Default **10**, hard upper
  bound **25** (author may set any value within `[1, 25]`). Applies to FR-007.
- **Q: What happens when the iteration or time limit is reached without a final answer?** → A:
  The agent finishes with an **error** terminal outcome, so non-convergence is explicit (rather
  than returning a partial answer). Applies to FR-007, FR-008, FR-013.
- **Q: Is the per-tool-call trace returned in the node output for this feature?** → A: **Yes**,
  the trace ("steps") is included in the output by default. Applies to FR-010.
- **Q: Does the optional tool allowlist ship in this feature?** → A: **Yes**, as an optional
  per-node allowlist; unset means all eligible tools are available. Applies to FR-012.
- **Q: Overall interaction time limit?** → A: Approximately **5 minutes** per agent interaction
  by default. Applies to FR-008.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Author an agent that solves a task using available tools (Priority: P1)

A workflow author drops an **AI agent node** into a workflow, gives it a task in plain language
(and optionally a system instruction), and connects its output to downstream nodes. At run time
the agent decides on its own which of the workflow engine's existing functions to call, calls
them, reads their results, and repeats until it has produced a final answer — which flows
downstream exactly like any other node's output.

**Why this priority**: This is the core value of the feature and the whole point of Phase B —
turning the existing function catalog into agent tools so an LLM can accomplish multi-step tasks
without the author wiring every branch by hand. Without it, nothing else matters.

**Independent Test**: Configure a workflow whose agent node is told to "add 2 and 3 and report
the result," with an arithmetic function available as a tool. Trigger the workflow and confirm
the final output contains the correct answer and that the arithmetic tool was actually invoked.

**Acceptance Scenarios**:

1. **Given** an agent node configured with a task and at least one eligible tool, **When** the
   workflow runs, **Then** the agent calls one or more tools and produces a final text answer as
   its output.
2. **Given** an agent node whose task needs no tools (e.g. "say hello"), **When** the workflow
   runs, **Then** the agent answers directly without calling any tool.
3. **Given** an agent node, **When** it is running its multi-step interaction, **Then** the
   engine's capacity to run other workflow nodes is not blocked for the duration of the
   interaction (the node completes asynchronously).
4. **Given** an agent node's final answer, **When** the agent finishes, **Then** its output is
   available to downstream nodes through normal edges, identical to any other node's output.

---

### User Story 2 - Bound and observe agent behaviour (Priority: P2)

An author needs the agent to be safe and debuggable: it must not loop forever, and the author
must be able to see what the agent did — which tools it called, with what arguments, and what
came back — to understand and trust the result.

**Why this priority**: An autonomous loop is only operable if it terminates predictably and is
observable. This makes the P1 capability trustworthy in real workflows, but the agent can
deliver value before this is fully polished.

**Independent Test**: Configure an agent with a low iteration limit against a task that would
otherwise loop, trigger it, and confirm the run terminates at the limit with a clear outcome and
that a step-by-step trace of tool calls is available in the run's recorded data.

**Acceptance Scenarios**:

1. **Given** an agent configured with a maximum number of reasoning iterations, **When** that
   limit is reached without a final answer, **Then** the agent stops and reports a terminal
   outcome (it does not loop indefinitely).
2. **Given** a completed agent run, **When** the author inspects the run, **Then** a trace is
   available listing each tool the agent called, the arguments it used, and the result or error.
3. **Given** an agent run that reports token usage from the model, **When** the run completes,
   **Then** the total token usage across all reasoning steps is available as part of the output.

---

### User Story 3 - Control which tools an agent may use (Priority: P3)

An author wants to limit an agent to a specific, safe set of tools, and trusts that risky or
incompatible functions are not silently handed to the model.

**Why this priority**: Sensible, safe defaults make the feature usable out of the box;
fine-grained per-node control is a refinement that increases confidence for production use.

**Independent Test**: Configure an agent that restricts itself to a named subset of tools,
trigger a task, and confirm the agent only ever calls tools from that subset; separately confirm
that functions known to be incompatible with the agent are never offered as tools regardless of
configuration.

**Acceptance Scenarios**:

1. **Given** an author who specifies an allowed subset of tools on an agent node, **When** the
   agent runs, **Then** only tools in that subset are made available to the model.
2. **Given** the catalog contains functions that are incompatible with in-loop tool use (those
   that complete asynchronously / are intercepted by the engine, or that accept free-form rather
   than declared parameters), **When** any agent runs, **Then** those functions are never offered
   to the model as tools.
3. **Given** the model requests a tool that does not exist or is not allowed, **When** the agent
   processes that request, **Then** the agent records the problem, informs the model, and
   continues rather than failing the whole run.

### Edge Cases

- **No eligible tools available**: the agent still runs and answers directly using only the model
  (no tools offered).
- **Model requests an unknown or disallowed tool**: the agent feeds an error back to the model as
  a tool result and continues the loop instead of crashing the run.
- **A tool returns an error**: the error is surfaced to the model as that tool's result so the
  model can recover or report it; the run is not aborted.
- **The model never converges**: the iteration limit guarantees termination with a clear terminal
  outcome.
- **The provider or model call fails**: the agent finishes with an error output that flows through
  the engine's normal error handling.
- **A tool would complete asynchronously**: such a tool is excluded from the agent's tool set; if
  one is ever invoked anyway, the agent treats it as unsupported rather than waiting forever.
- **Total interaction exceeds a reasonable time bound**: the agent enforces an overall time limit
  and finishes with a terminal outcome.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST provide an AI agent workflow node that accepts a task (and optional
  system instruction, model selection, and provider selection) and produces a final text answer
  as its output.
- **FR-002**: The agent MUST reason in a loop: request the model, perform any tool calls the
  model asks for, feed the results back to the model, and repeat until the model produces a final
  answer or a configured iteration limit is reached.
- **FR-003**: The agent MUST derive its available tools from the workflow engine's existing
  catalog of functions, using each function's declared input parameters to describe the tool to
  the model — without authors having to define a separate tool catalog.
- **FR-004**: The agent MUST only offer functions that are compatible with in-loop use. It MUST
  exclude functions that complete asynchronously or are intercepted by the engine, and functions
  that accept free-form (undeclared) parameters.
- **FR-005**: When the model requests a tool, the agent MUST invoke the corresponding function,
  obtain its result inline, and return that result to the model as the corresponding tool's
  output before continuing.
- **FR-006**: The agent MUST complete asynchronously so that long, multi-step interactions do not
  occupy engine execution capacity for the duration of the interaction.
- **FR-007**: The agent MUST enforce a configurable maximum number of reasoning iterations with a
  safe default and an upper bound, and MUST terminate with a clear terminal outcome when the limit
  is reached.
- **FR-008**: The agent MUST enforce an overall time limit on the whole interaction and terminate
  with a clear terminal outcome if exceeded.
- **FR-009**: The agent MUST handle a request for an unknown or disallowed tool, and a tool that
  returns an error, by recording the problem and continuing the loop rather than aborting the run.
- **FR-010**: The agent MUST record a per-run trace of the tools it called, the arguments it used,
  and each tool's result or error, available for inspection after the run.
- **FR-011**: The agent MUST report aggregated model token usage across all reasoning steps as
  part of its output when the provider supplies it.
- **FR-012**: Authors MUST be able to optionally restrict an agent node to an allowed subset of
  tools; when unset, all eligible tools are available.
- **FR-013**: On any terminal outcome (success, model/provider failure, iteration or time limit),
  the agent MUST report exactly one result so the run always completes deterministically.
- **FR-014**: An agent's final answer MUST flow to downstream nodes through the engine's normal
  output/edge mechanism, identical to any other node.
- **FR-015**: A required task input MUST be validated; if it is missing the node MUST fail fast
  with a clear error before starting any model interaction.

### Key Entities *(include if feature involves data)*

- **AI Agent Node**: A workflow node configured with a task, optional system instruction, model,
  provider, iteration limit, and optional tool allowlist; produces a final answer, a usage
  summary, and a trace.
- **Tool**: A description of an existing engine function made available to the model, derived from
  the function's name and declared input parameters.
- **Tool Call**: A model-issued request to run a specific tool with specific arguments.
- **Reasoning Step / Trace Entry**: A record of one tool call within a run — the tool, its
  arguments, and its result or error.
- **Conversation**: The accumulating sequence of messages (instruction, task, model replies, tool
  results) that drives the loop.
- **Usage Summary**: Aggregated token counts for the run.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An author can configure an agent node and have it complete a task that requires at
  least one tool call, end-to-end, with no custom code beyond node configuration.
- **SC-002**: For a task requiring N tool calls, the agent calls the correct tools and returns a
  final answer reflecting their results, demonstrated for at least a single-tool and a two-tool
  task.
- **SC-003**: 100% of agent runs terminate — every run ends in a final answer or a clear terminal
  outcome within the configured iteration and time limits; no run loops indefinitely.
- **SC-004**: While an agent node is running its multi-step interaction, other workflow nodes
  continue to be executed by the engine (the agent does not reduce concurrent execution capacity
  for the duration of its interaction).
- **SC-005**: After any run, the author can see a complete trace of every tool the agent called
  with arguments and results, and an aggregated token-usage figure.
- **SC-006**: Functions known to be incompatible with in-loop use are never offered to the model,
  verified across the engine's built-in function set.
- **SC-007**: A malformed or failing tool interaction (unknown tool, tool error) never aborts the
  whole run; the agent records it and continues or finishes cleanly.

## Assumptions

- **Default iteration limit**: 10 reasoning iterations by default, with a hard upper bound of 25,
  unless the author overrides it (to be confirmed in clarification).
- **Terminal outcome on limit**: reaching the iteration or time limit produces an **error**
  terminal outcome (rather than returning the last partial answer), to make non-convergence
  explicit (to be confirmed in clarification).
- **Overall time limit**: a single agent interaction is bounded at roughly 5 minutes by default.
- **Trace output**: the per-run trace ("steps") is included in the node output by default for
  observability.
- **Tool allowlist**: optional; an unset allowlist means all eligible tools are available.
- **Provider/model defaults**: when provider or model is unset, the agent uses the engine's
  configured default provider/model, consistent with the existing chat node.
- **Scope boundary (Phase B)**: only synchronous, declared-parameter functions are exposed as
  tools. Asynchronous tools, streaming, a native Anthropic provider, conversation memory across
  runs, and an autonomous "agent owns the whole flow" mode are explicitly out of scope for this
  feature and tracked separately.
