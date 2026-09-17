---
name: capability-registry-and-agents
description: >-
  Maps what FUSE's agent machinery actually is today — ai/agent and ai/chat's real parameters, the provider
  registry, token accounting, structured output, the tool-catalog adapter — and the three structural gaps the
  capability registry must close: node/tool projection parity, per-execution addressing of node output, and
  durable human gates. Use when working B-03, B-04 or any Tier C agent task, or before assuming an agent tool
  call is traced and journalled like a node call.
---

# capability-registry-and-agents

Agents now run **inside the engine**. That is what promotes A-09, A-10 and A-11 into Tier A, and it
is why the capability registry (B-03) has to be designed before either projection is built.

Backlog ids are from the repo-root backlog (`BACKLOG_V2.md`). *If it has been renamed to
`BACKLOG.md`, the canonical file is the one titled "FUSE — product shape and backlog."*

> **§ TODAY** = shipped behaviour, `file:line` traceable. **§ PROPOSED** = backlog design that does
> not exist. Never quote a § PROPOSED option as engine behaviour.

## When to use

- **B-03** capability registry, **B-04** MCP ingestion.
- **C-01** agent↔registry tool binding, **C-02** memory, **C-04** GenAI telemetry, **C-05** human
  gate payloads, **C-06/C-07** durable timers and awakeable IDs.
- **A-11** per-execution addressing of node output — the single most expensive item in the backlog
  to reverse.
- The wire half of a registered capability lives in `.agents/skills/remote-node-protocol/SKILL.md`.

## Ground truth

| File | Authoritative for |
| --- | --- |
| `internal/packages/functions/ai/agent.go` | The whole reasoning loop and `ai/agent`'s declared parameters |
| `internal/packages/functions/ai/chat.go` | `ai/chat`, provider resolution, temperature parsing |
| `internal/packages/functions/ai/tools.go` | The `ToolRegistry` port + function-metadata→JSON-Schema conversion |
| `internal/packages/agent_tools.go` | The adapter that turns the package registry into the tool catalog |
| `pkg/llm/provider.go` / `registry.go` | The entire public LLM contract and the per-environment registry |
| `internal/app/di/llm.go` | The only place providers are named and enabled |
| `internal/metrics/registry.go` | Where LLM usage lands, and its label set |
| `internal/workflow/workflow.go` | `aggregatedOutput` producer/consumer — the A-11 crux |
| `pkg/store/kv.go` | The objx dot-path KV that *is* `aggregatedOutput` |
| `internal/workflow/awakeable.go` + `internal/handlers/resolve_awakeable.go` | The human-gate primitives |
| `docs/adr/0007`, `0026`, `0027`, `0028`, `0029`, `0030` | Tools-from-functions, agent-as-orchestrator (Proposed), async tool invocation (Proposed), context/memory, cost, structured output |

---

## § TODAY — the parity question, answered

**B-03's hard design requirement:** an agent invoking a capability and a graph node invoking the same
capability must produce the same journal entries, the same span shape and the same idempotency
behaviour.

**That parity is FALSE today — by construction, not by omission.** This is the single most
decision-relevant fact in this pack.

| | Graph node path | Agent tool path |
| --- | --- | --- |
| Entry point | `internal/actors/workflow_func.go:63 HandleMessage` | `ai/agent.go:271 e.tools.InvokeTool(...)` |
| Adapter | — | `internal/packages/agent_tools.go:79 InvokeTool` (linear scan of `registry.List()`) |
| Execution | `LoadedPackage.ExecuteFunction` → `InternalFunctionTransport.Execute` | `LoadedPackage.ExecuteFunctionSync` (`loaded_package.go:50`) → `ExecuteSync` (`internal.go:68`) |
| Span | `node.execute` created at `workflow_func.go:90` | **none** |
| Journal | `step:started` at dispatch (`newRunFunctionAction`, `workflow.go:616`; also `:171` on trigger, `:488` on manual retry) + `step:completed`/`step:failed` via `SetResultFor` (`workflow.go:422`) | **none** |
| Metric | `NodeExecDuration{function_id,status}` via `recordNodeDuration` (`workflow_func.go:181`) | **none** |
| Retry / timeout | node `RetryPolicy` + `ExecutionTimer` | **none** — one inline call, no timeout of its own |
| Idempotency | none (trigger-level only) | none |

The tool path deliberately never reaches the actor system, so it misses the sole `node.execute` span
site and the sole `step:completed` producer. **The only durable record of a tool call is the `steps`
array inside the agent node's single `FunctionOutput`**, which lands in one `step:completed` journal
entry as an opaque blob (`agent.go:148` — `steps` is a local variable; each entry is built at
`agent.go:282` on success and `:288` on failure).

Achieving parity therefore requires either journalling from inside the agent goroutine — which today
has no journal handle, because `ExecutionInfo` carries only `WorkflowID`/`ExecID`/`Environment`/
`Input`/`Finish` — or routing tool calls back through the actor system. ADR-0027 (Proposed) names the
second and constrains how: a **second typed per-execution port** (`ExecRuntime` with
`InvokeAsync(...)`), injected into orchestrating nodes, and explicitly **not** a field on
`workflow.ExecutionInfo` (the earlier `Handle any` field was removed for exactly that reason).

⚠️ **The tool-eligibility predicate is half-dead.** `isExposableTool`
(`internal/packages/agent_tools.go:95`) requires `fn.Transport != nil && fn.Metadata.Transport ==
transport.Internal`. But `MapToRegistryPackage` hardcodes `Transport: transport.Internal` into every
registry-side metadata (`internal/packages/loaded_package.go:70`), so the second clause is vacuously
true and the real discriminator is the nil-transport check. A B-03 predicate written against
`Metadata.Transport` will match everything.

The catalog also excludes, by construction: the whole `ai` package (`agent_tools.go:53`), any
`CustomParameters` function, and the `interceptedOrAsyncFunctionIDs` denylist (`agent_tools.go:18-24`:
`system/sleep`, `system/wait`, `system/subworkflow`, `system/foreach`, `fuse/pkg/logic/timer`).
An agent can therefore not open a human gate, sleep, spawn a sub-workflow or call another agent —
the exact primitives "agents in Core" needs. A tool that returns `Async` at runtime is rejected
inline with `"tool is asynchronous and not supported by the agent"` (`agent.go:276`). Tool
descriptions are a placeholder: `fmt.Sprintf("FUSE function %s", fullID)` (`agent_tools.go:67`).

## § TODAY — what `ai/agent` and `ai/chat` actually are

Both are ordinary internal functions registered by `ai.New(providers, tools, usage)`
(`internal/packages/functions/ai/package.go:18`, `PackageID = "fuse/pkg/ai"`), both **async**: they
return `NewFunctionResultAsync()` and complete via `execInfo.Finish` from a goroutine with its own
`context.WithTimeout(context.Background(), …)` — `agentTimeout = 5 * time.Minute` (`agent.go:26`),
`chatTimeout = 2 * time.Minute` (`chat.go:18`).

| | `ai/agent` (`agent.go:33`) | `ai/chat` (`chat.go:24`) |
| --- | --- | --- |
| Inputs | `input`(required), `provider`, `model`, `systemPrompt`, `temperature`, `maxIterations`, `allowedTools`, `maxContextTokens`, `contextStrategy`, `outputSchema` | `input`(required), `provider`, `model`, `systemPrompt`, `temperature`, `outputSchema` |
| Outputs | `output`, `usage`, `steps` | `output`, `usage` |
| Loop bounds | `defaultMaxIterations = 10`, `maxMaxIterations = 25` (`agent.go:22-24`) | single completion |

**Provider abstraction.** `pkg/llm/provider.go:82` — `Provider{Name(); Chat(ctx, ChatRequest)}`. That
is the whole contract: no capability metadata, no pricing. `StreamingProvider` (`:101`) exists with
**zero implementations** — `ChatStream` appears only in the interface declaration. Resolution is
per-`(environment, name)` through `pkg/llm/registry.go`'s `ProviderFactory func(ctx, environment)`,
so ADR-0031 secret refs resolve per execution; `resolveProvider` (`chat.go:178`) falls back to the
registry default. Providers are named at `internal/app/di/llm.go:19-23` and registered only by
`provideLLMRegistry` (`llm.go:45`) — `openai`, `openrouter`, `ollama`, `gemini`, `anthropic`,
**all disabled by default** (`LLMProviderConfig.Enabled`, `config.go:66`, `envDefault:"false"`);
with none enabled it logs `no LLM providers enabled; ai/chat and ai/agent nodes will be unavailable`
(`llm.go:77`).

⚠️ `LLM_<PROVIDER>_TEMPERATURE` is **dead config**: declared at `internal/app/config/config.go:73`
and read nowhere. The only temperature that reaches a request is the per-node `temperature` input
via `optionalTemperature` (`chat.go:186`).

**Usage and cost.** `ai.UsageRecorder` (`ai/usage.go:8`) is a two-method port —
`RecordUsage(function, provider, model, llm.Usage)` and `RecordCall(function, provider, model, status)`
— adapted in `internal/packages/usage_recorder.go` onto Prometheus counters
(`internal/metrics/registry.go:67-77`):

```
fuse_llm_tokens_total{function, provider, model, type}
fuse_llm_calls_total{function, provider, model, status}
```

⚠️ **No workflow id, exec id, node id, tenant or environment label exists on either counter**, and
the recorder writes nowhere else. "What did this run cost" is unanswerable from any FUSE surface
today — that is **C-04**, which is additionally blocked by `workflow_func.go:96` (`_ = nodeCtx`
discards the span context) and by the node span ending at `:166`, before an async node's first LLM
call.

**Structured output** (ADR-0030, `ai/structured.go`): a synthetic `respond` tool forced with
`ToolChoice: "required"`, up to `structuredRepairAttempts = 2`. Validation is deliberately lenient
(`valueMatchesType`, `:125`): unknown type strings pass, extra fields are never rejected, and
`parseOutputSchema` (`:20`) returns nil on a malformed schema — silently degrading the node to
free-form text rather than erroring.

**Context bounding** (ADR-0028, `ai/context.go`): `estimateTokens` is `len(chars)/4`
(`approxCharsPerToken = 4`, `:13`) — a heuristic, not a tokenizer. Strategies are `drop-oldest`
(default) and `summarize` (an extra `provider.Chat`). It trims **inside one node execution only**.

## § TODAY — the state problem (A-11's crux)

`aggregatedOutput` is a `*store.KV` created at `internal/workflow/workflow.go:65`.

- **Producer:** `SetResultFor` → `w.aggregatedOutput.Set(entry.FunctionNodeID, result.Output.Data)`
  (`workflow.go:428`) — keyed by **node ID**.
- **Consumer:** `applyFlowMapping` → `w.aggregatedOutput.Get(mapping.Variable)` (`workflow.go:751`),
  where `mapping.Variable` is `"<nodeID>.<outputParam>"`, split by `strutil.AfterFirstDot`
  (`workflow.go:728`).
- **Backing store:** `pkg/store/kv.go` over `objx` — `Set` (`:72`) and `Get` (`:79`) use **dot-path**
  semantics; `Snapshot` (`:44`) is `maps.Clone`, a **shallow** copy.

Three consequences any A-11 design must address, not just the re-keying:

1. **A node that executes twice overwrites its own prior output.** The second write wins; there is no
   history in the data plane, though the journal keeps every execution.
2. **A node id containing a `.` is silently written as a nested key** (objx dot notation), and
   `AfterFirstDot` splits the mapping variable on the *first* dot — so a dotted node id also breaks
   output-parameter validation.
3. **A retained snapshot shares the inner per-node maps with the live KV**, because `Snapshot` is
   shallow. Its consumers are the execution snapshot, sub-workflow completion, and repository `Save`.

**Why an agent loop makes this fatal:** an agent loop is a cycle; every turn re-executes the same
node; each turn destroys the previous turn's output in the data plane. Conversation history, per-turn
selections, and any "reject item 2, keep 1, 3 and 4" correction are unaddressable from inside the
graph. Compounding it, the transcript itself never leaves the goroutine: `messages`, `totalUsage` and
`steps` are locals in `agentExecutor.run` (`agent.go:146-148`) — confirmed by reading — so nothing
about a turn is durable until the node finishes, and no run can be resumed mid-conversation.

⚠️ *Inference, not verified — do not build on it.* What a restart **during** an agent turn actually
does is unsettled. An async node is journalled `step:completed` at dispatch (see
`.agents/skills/remote-node-protocol/SKILL.md`, § TODAY), so `findPendingThreads` may not treat the
agent node as pending and may not re-dispatch it at all — which would lose the turn silently rather
than repeat it. Settle it with the **F-02** harness before designing around either outcome.

### § PROPOSED — A-11's two options (neither exists; an ADR precedes either)

| Option | Shape | What it costs |
| --- | --- | --- |
| **(a) Turn state as a first-class Core concept** — *the backlog leans here* | A dedicated store addressed by `(runID, agentNodeID, turn)`, entirely separate from `aggregatedOutput` | A new repository beside `internal/repositories/awakeable.go`, new `JournalEntryType` constants **plus** an `ALTER TYPE journal_entry_type ADD VALUE` migration (pattern: `000004_add_manual_retry_journal_type.up.sql`), and a way for the agent goroutine to reach it — which today it cannot. Smaller blast radius, no breaking schema change, and it **overlaps C-02 memory**, so do them together |
| **(b) Re-key `aggregatedOutput` by `(nodeID, execID)`** with a `latest` alias | Correct and general | Breaking-ish; needs a migration story and must also settle the dot-path, `AfterFirstDot` and shallow-snapshot problems above |

**Do not start by re-keying the KV.** Write the ADR first (`.agents/skills/write-adr/SKILL.md`).
This is the one item in the backlog that is expensive to reverse.

## § TODAY — human-gate primitives, and what is missing

| Primitive | Location | State |
| --- | --- | --- |
| `system/wait` metadata | `internal/packages/functions/system/wait.go` | inputs `timeout`, `filter`; outputs `data`, `timedOut`. Body is a placeholder — intercepted by the handler |
| Awakeable row | `internal/workflow/awakeable.go:24` | `{ID, WorkflowID, ExecID, ThreadID, CreatedAt, Timeout, DeadlineAt, Status, Result}` |
| Mint | `internal/actors/workflow_handler.go:728` (`uuid.New()`), journalled at `:785` as `Data{"awakeableId", "timeout"}` | |
| Deadline | `workflow_handler.go:775` — `DeadlineAt: now.Add(action.Timeout)` | ⚠️ **always set**, so a no-timeout gate gets `DeadlineAt == CreatedAt`. A C-06 sweeper treating `deadline_at <= now()` as expired would kill every untimed gate |
| Resolve | `POST /v1/awakeables/{awakeableID}/resolve` — `internal/handlers/resolve_awakeable.go:55` | The **only** awakeable route (`mux_worker.go:270`). It 404s on missing and 400s on non-pending — the validation template the async-exec route lacks |

⚠️ **The `timedOut` output is never true.** Grep finds exactly three hits: the metadata declaration
(`wait.go:24`), a test, and the literal `false` at `workflow_handler.go:845`. `AwakeableTimedOut` and
`AwakeableCancelled` are declared and **never assigned** in production code. A timed-out gate goes
through `handleMsgTimeout` (`:551`) into a generic `FunctionError{"execution timeout exceeded"}`,
indistinguishable from a crashed node, and the row stays `pending` forever.

⚠️ **No external caller can learn a pending gate's awakeable id** (**C-07**). It exists only in a
journal entry's `Data` map; `SnapshotTimelineEvent` (`internal/workflow/execution_snapshot.go:47`) has
no `Data` field; `BuildTrace` handles only `step:*` and `state:changed`
(`internal/workflow/trace_builder.go:22-68`); snapshot and trace persist only from
`sendWorkflowCompleted` (`workflow_handler.go:518-519`), i.e. at terminal state; `GET /v1/workflows/{id}`
returns `dtos.GetWorkflowResponse{workflowId, status}` only; and
`AwakeableRepository.FindPending` (`internal/repositories/awakeable.go:16`) has **no production
caller** — only the two implementations and their tests.

**C-05** (typed gate payloads: options, confirm, form) and **C-06** (durable timers and an expiry
sweeper — every timer today is a per-actor `SendAfter`, lost on restart) build on these. C-05's
"reject one item, keep the rest" semantics depend on A-11 being resolved, because per-item history
has nowhere to live.

## § PROPOSED — B-04 MCP ingestion (nothing exists; no MCP reference anywhere in engine code)

An MCP tool maps cleanly onto `ai.ToolDescriptor{FunctionID, MangledName, Description, Parameters}`
(`ai/tools.go:32`), and `ParameterSchemaToJSONSchema` (`:68`) already converts in the other
direction. **The blocker is execution, not description:** an ingested MCP tool has no code-backed
function, so `MapToRegistryPackage` registers it metadata-only with a nil `Transport`, and
`isExposableTool` rejects it — visible and un-callable until the remote transport (**B-02**) exists.
Name mangling (`MangleToolName`, `/` → `__`, `ai/tools.go:55`) also needs a rule for
server-qualified MCP names.

**Ingestion beats exposure for now**: real customer estates already speak MCP, so importing their
tools is how they arrive on FUSE without a rewrite; exposing FUSE capabilities *as* an MCP server has
no customer behind it yet. Do not build both at once.

## Traps

| Trap | Evidence |
| --- | --- |
| Assuming an agent tool call is traced/journalled like a node call | The parity table above |
| Writing a registry predicate against `Metadata.Transport` | Hardcoded `Internal` at `loaded_package.go:70` |
| Assuming the agent transcript is durable | `messages`/`totalUsage`/`steps` are locals in `agent.go:146-148` |
| Assuming token cost is attributable to a run | `fuse_llm_*` labels are `{function, provider, model, type\|status}` only |
| Assuming `maxContextTokens` is a real token budget | `len(chars)/4`, `ai/context.go:13` |
| Assuming `outputSchema` enforces types | `valueMatchesType` accepts unknown type strings; a malformed schema silently disables the feature |
| Assuming an agent can wait, sleep or spawn a sub-workflow | `interceptedOrAsyncFunctionIDs`, `agent_tools.go:18-24` |
| Adding a sync, non-`CustomParameters` function anywhere | It is **automatically** offered to every agent as a tool; denylist it in the same PR |
| `AgentToolRegistry` cost at scale | `ListTools` rebuilds every descriptor and JSON schema on **every** agent execution; `InvokeTool` linear-scans `registry.List()` on **every** tool call |

## Commands (confirmed to exist)

```bash
make swagger                                       # REQUIRED on a fresh clone before build/test
make lint && make build && make test               # the mandated gate, in this order
go test ./internal/packages/functions/ai/...       # agent, chat, tools, context, structured — pure unit tests, no actors
go test -run TestAgent ./internal/packages/functions/ai/
go test ./internal/packages/ -run 'ListTools|InvokeTool|isExposableTool'
go test ./pkg/llm/... ./internal/llm/...
go test ./internal/workflow/ -run 'BuildTrace|BuildExecutionSnapshot|Journal'
make migrate                                       # ./bin/fuse migrate — needed before any journal-enum change
make seed                                          # ./bin/fuse seed examples -l debug
make infra-up                                      # Postgres + S3 + etcd; engine on the host
LLM_OLLAMA_ENABLED=true LLM_OLLAMA_BASE_URL=http://localhost:11434/v1 \
  LLM_OLLAMA_MODEL=<model> LLM_DEFAULT_PROVIDER=ollama make run   # the no-API-key path
curl -s localhost:9090/metrics | grep fuse_llm_    # the only cost surface that exists
```

⚠️ `./bin/fuse seed examples --ci` skips `github-request-example.json` and any file containing
`fuse/pkg/logic/timer` **or** `fuse/pkg/ai/` (`internal/app/cli/seed.go:184`), so the five `ai-*`
example schemas do not run in CI. There is no `make examples-ci` target; `--ci` is the only switch.

$ARGUMENTS
