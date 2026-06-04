package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/open-source-cloud/fuse/internal/packages/transport"
	"github.com/open-source-cloud/fuse/pkg/llm"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/rs/zerolog/log"
)

// AgentFunctionID is the id of the agent function.
const AgentFunctionID = "agent"

const (
	// defaultMaxIterations is the default reasoning-loop bound.
	defaultMaxIterations = 10
	// maxMaxIterations is the hard upper bound an author may request.
	maxMaxIterations = 25
	// agentTimeout bounds the whole multi-step interaction.
	agentTimeout = 5 * time.Minute
)

// ErrAgentInputRequired is returned when the required task input is missing.
var ErrAgentInputRequired = errors.New("ai/agent: input is required")

// AgentFunctionMetadata returns the metadata for the agent function.
func AgentFunctionMetadata() workflow.FunctionMetadata {
	return workflow.FunctionMetadata{
		Transport: transport.Internal,
		Input: workflow.InputMetadata{
			CustomParameters: false,
			Parameters: []workflow.ParameterSchema{
				{Name: "input", Type: "string", Required: true, Description: "The task / goal the agent should accomplish"},
				{Name: "provider", Type: "string", Required: false, Description: "Provider registry key (e.g. openai, ollama). Defaults to the configured default provider"},
				{Name: "model", Type: "string", Required: false, Description: "Model id. Defaults to the provider's configured default model"},
				{Name: "systemPrompt", Type: "string", Required: false, Description: "Optional system instruction prepended to the conversation"},
				{Name: "temperature", Type: "float", Required: false, Description: "Sampling temperature; if omitted the provider default is used"},
				{Name: "maxIterations", Type: "int", Required: false, Default: defaultMaxIterations, Description: "Maximum reasoning iterations (clamped to [1, 25])"},
				{Name: "allowedTools", Type: "array", Required: false, Description: "Optional allowlist of full function ids the agent may use as tools; empty means all eligible tools"},
				{Name: "maxContextTokens", Type: "int", Required: false, Description: "Optional token budget for the running transcript (approximate); 0/absent disables trimming (ADR-0028)"},
				{Name: "contextStrategy", Type: "string", Required: false, Description: "When over maxContextTokens: 'drop-oldest' (default) or 'summarize' (an extra LLM call summarizes dropped turns)"},
				{Name: "outputSchema", Type: "array", Required: false, Description: "Optional list of {name,type,required,description} fields; when set the final output is a validated object matching this schema (ADR-0030)"},
			},
			Edges: workflow.InputEdgeMetadata{
				Count:      0,
				Parameters: make([]workflow.ParameterSchema, 0),
			},
		},
		Output: workflow.OutputMetadata{
			Parameters: []workflow.ParameterSchema{
				{Name: "output", Type: "string", Required: true, Description: "The agent's final text answer"},
				{Name: "usage", Type: "map", Required: false, Description: "Aggregated token usage across all reasoning steps"},
				{Name: "steps", Type: "array", Required: false, Description: "Trace of each tool call: tool, arguments, and result or error"},
			},
			Edges: make([]workflow.OutputEdgeMetadata, 0),
		},
	}
}

// makeAgentFunction builds the ai/agent function, closing over the provider
// registry, the tool registry, and the usage recorder (ADR-0029).
func makeAgentFunction(providers llm.Registry, tools ToolRegistry, usage UsageRecorder) workflow.Function {
	return func(execInfo *workflow.ExecutionInfo) (workflow.FunctionResult, error) {
		input := execInfo.Input

		userInput := input.GetStr("input")
		if userInput == "" {
			return workflow.NewFunctionResultError(ErrAgentInputRequired)
		}

		providerName := input.GetStr("provider")

		llmTools, byMangled := buildTools(tools.ListTools(), allowedToolSet(input))

		executor := &agentExecutor{
			tools:       tools,
			byMangled:   byMangled,
			llmTools:    llmTools,
			model:       input.GetStr("model"),
			temp:        optionalTemperature(input),
			maxIters:    clampIterations(input.GetInt("maxIterations")),
			wfID:             execInfo.WorkflowID,
			execID:           execInfo.ExecID,
			environment:      execInfo.Environment,
			usage:            usage,
			maxContextTokens: input.GetInt("maxContextTokens"),
			contextStrategy:  contextStrategyOrDefault(input.GetStr("contextStrategy")),
			outputSchema:     parseOutputSchema(input),
		}

		messages := make([]llm.Message, 0, 4)
		if systemPrompt := input.GetStr("systemPrompt"); systemPrompt != "" {
			messages = append(messages, llm.Message{Role: llm.RoleSystem, Content: systemPrompt})
		}
		messages = append(messages, llm.Message{Role: llm.RoleUser, Content: userInput})

		// Provider resolution and the reasoning loop run in their own goroutine and report back
		// via Finish so the WorkflowFunc pool worker is freed immediately (mirrors ai/chat).
		// Resolution is here too because per-context provider keys (ADR-0031) may hit the secret
		// store. The provider is resolved ONCE and reused across the loop (stable within a run).
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), agentTimeout)
			defer cancel()

			provider, err := resolveProvider(ctx, providers, execInfo.Environment, providerName)
			if err != nil {
				log.Error().Err(err).Str("provider", providerName).Msg("ai/agent provider resolution failed")
				execInfo.Finish(errorOutput(fmt.Sprintf("ai/agent: provider resolution failed: %v", err)))
				return
			}
			executor.provider = provider

			execInfo.Finish(executor.run(ctx, messages))
		}()

		return workflow.NewFunctionResultAsync(), nil
	}
}

// agentExecutor holds the immutable per-run parameters for the reasoning loop.
type agentExecutor struct {
	provider    llm.Provider
	tools       ToolRegistry
	byMangled   map[string]string // mangled tool name -> real function id
	llmTools    []llm.Tool
	model       string
	temp        *float32
	maxIters    int
	wfID             workflow.ID
	execID           workflow.ExecID
	environment      string
	usage            UsageRecorder
	maxContextTokens int
	contextStrategy  string
	outputSchema     []workflow.ParameterSchema
}

// run drives the reasoning loop until a final answer, an error, or the iteration
// limit. It returns exactly one FunctionOutput; the caller calls Finish once.
func (e *agentExecutor) run(ctx context.Context, messages []llm.Message) workflow.FunctionOutput {
	totalUsage := llm.Usage{}
	steps := make([]map[string]any, 0)

	for i := 0; i < e.maxIters; i++ {
		// Bound the growing transcript to the configured token budget (ADR-0028).
		if trimmed, step := e.applyContextPolicy(ctx, messages); step != nil {
			messages = trimmed
			steps = append(steps, step)
		}

		resp, err := e.provider.Chat(ctx, llm.ChatRequest{
			Model:       e.model,
			Messages:    messages,
			Tools:       e.llmTools,
			Temperature: e.temp,
			ToolChoice:  "auto",
		})
		if err != nil {
			e.usage.RecordCall(AgentFunctionID, e.provider.Name(), e.model, "error")
			log.Error().Err(err).Str("provider", e.provider.Name()).Msg("ai/agent completion failed")
			return errorOutput(fmt.Sprintf("ai/agent: completion failed: %v", err))
		}
		e.usage.RecordCall(AgentFunctionID, e.provider.Name(), e.model, "success")
		e.usage.RecordUsage(AgentFunctionID, e.provider.Name(), e.model, resp.Usage)

		totalUsage.PromptTokens += resp.Usage.PromptTokens
		totalUsage.CompletionTokens += resp.Usage.CompletionTokens
		totalUsage.TotalTokens += resp.Usage.TotalTokens

		messages = append(messages, resp.Message)

		if len(resp.Message.ToolCalls) == 0 {
			// Structured output (ADR-0030): coerce the final answer into the requested schema.
			if len(e.outputSchema) > 0 {
				obj, u, serr := structuredOutput(ctx, e.provider, e.model, messages, e.outputSchema, e.usage, AgentFunctionID)
				if serr != nil {
					return errorOutput(fmt.Sprintf("ai/agent: structured output failed: %v", serr))
				}
				totalUsage.PromptTokens += u.PromptTokens
				totalUsage.CompletionTokens += u.CompletionTokens
				totalUsage.TotalTokens += u.TotalTokens
				return successOutputData(obj, totalUsage, steps)
			}
			return successOutput(resp.Message.Content, totalUsage, steps)
		}

		for _, tc := range resp.Message.ToolCalls {
			toolMsg, step := e.executeToolCall(tc)
			messages = append(messages, toolMsg)
			steps = append(steps, step)
		}
	}

	return errorOutput("ai/agent: max iterations reached")
}

// applyContextPolicy bounds the running transcript to maxContextTokens (ADR-0028). It returns the
// trimmed messages and a step record, or (nil, nil) when no trimming was needed.
func (e *agentExecutor) applyContextPolicy(ctx context.Context, messages []llm.Message) ([]llm.Message, map[string]any) {
	if e.maxContextTokens <= 0 {
		return nil, nil
	}
	kept, dropped := trimContext(messages, e.maxContextTokens)
	if len(dropped) == 0 {
		return nil, nil
	}
	step := map[string]any{"context": "trimmed", "strategy": e.contextStrategy, "droppedTurns": len(dropped)}
	if e.contextStrategy == contextStrategySummarize {
		if summary := e.summarizeDropped(ctx, dropped); summary != "" {
			h := headLen(messages)
			summaryTurn := llm.Message{Role: llm.RoleSystem, Content: "Summary of earlier turns: " + summary}
			out := concatMessages(messages[:h], append([]llm.Message{summaryTurn}, kept[h:]...))
			step["summarized"] = true
			return out, step
		}
	}
	return kept, step
}

// summarizeDropped asks the provider to summarize the dropped turns into one note. Returns "" on
// error, so the caller falls back to plain drop-oldest.
func (e *agentExecutor) summarizeDropped(ctx context.Context, dropped []llm.Message) string {
	var b strings.Builder
	for _, m := range dropped {
		fmt.Fprintf(&b, "%s: %s\n", m.Role, m.Content)
	}
	resp, err := e.provider.Chat(ctx, llm.ChatRequest{
		Model: e.model,
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "Summarize the following conversation turns concisely, preserving facts, tool results, and decisions."},
			{Role: llm.RoleUser, Content: b.String()},
		},
	})
	if err != nil {
		log.Warn().Err(err).Msg("ai/agent context summarization failed; dropping oldest turns instead")
		return ""
	}
	e.usage.RecordCall(AgentFunctionID, e.provider.Name(), e.model, "success")
	e.usage.RecordUsage(AgentFunctionID, e.provider.Name(), e.model, resp.Usage)
	return resp.Message.Content
}

// executeToolCall resolves, invokes, and records a single model-requested tool
// call. It never aborts the run: unknown tools, bad arguments, invocation errors,
// and tool errors are all fed back to the model as the tool's result so it can
// recover or report them.
func (e *agentExecutor) executeToolCall(tc llm.ToolCall) (llm.Message, map[string]any) {
	realID, known := e.byMangled[tc.Name]
	if !known {
		return e.toolError(tc, tc.Name, nil, fmt.Sprintf("unknown or disallowed tool %q", tc.Name))
	}

	var args map[string]any
	if len(tc.Arguments) > 0 {
		if err := json.Unmarshal(tc.Arguments, &args); err != nil {
			return e.toolError(tc, realID, nil, fmt.Sprintf("invalid tool arguments: %v", err))
		}
	}

	nestedInput, err := workflow.NewFunctionInputWith(args)
	if err != nil {
		return e.toolError(tc, realID, args, fmt.Sprintf("failed to build tool input: %v", err))
	}

	result, err := e.tools.InvokeTool(realID, workflow.NewExecutionInfo(e.wfID, e.execID, e.environment, nestedInput))
	if err != nil {
		return e.toolError(tc, realID, args, err.Error())
	}
	if result.Async {
		return e.toolError(tc, realID, args, "tool is asynchronous and not supported by the agent")
	}
	if result.Output.Status == workflow.FunctionError {
		return e.toolError(tc, realID, args, fmt.Sprintf("tool returned an error: %v", result.Output.Data))
	}

	step := map[string]any{"tool": realID, "arguments": args, "result": result.Output.Data}
	return toolMessage(tc, result.Output.Data), step
}

// toolError builds the tool-result message and trace step for a failed tool call.
func (e *agentExecutor) toolError(tc llm.ToolCall, toolID string, args map[string]any, msg string) (llm.Message, map[string]any) {
	step := map[string]any{"tool": toolID, "error": msg}
	if args != nil {
		step["arguments"] = args
	}
	return toolMessage(tc, map[string]any{"error": msg}), step
}

// buildTools converts tool descriptors into llm.Tool definitions (optionally
// filtered by an allowlist of real function ids) and the mangled->real id map.
func buildTools(descriptors []ToolDescriptor, allowed map[string]struct{}) ([]llm.Tool, map[string]string) {
	tools := make([]llm.Tool, 0, len(descriptors))
	byMangled := make(map[string]string, len(descriptors))
	for _, d := range descriptors {
		if allowed != nil {
			if _, ok := allowed[d.FunctionID]; !ok {
				continue
			}
		}
		tools = append(tools, llm.Tool{Name: d.MangledName, Description: d.Description, Parameters: d.Parameters})
		byMangled[d.MangledName] = d.FunctionID
	}
	return tools, byMangled
}

// allowedToolSet reads the optional allowedTools input into a set of full
// function ids, or nil when unset (meaning all eligible tools are allowed).
func allowedToolSet(input *workflow.FunctionInput) map[string]struct{} {
	raw := input.GetAnySliceOrDefault("allowedTools", nil)
	if len(raw) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok && s != "" {
			set[s] = struct{}{}
		}
	}
	if len(set) == 0 {
		return nil
	}
	return set
}

// clampIterations applies the default and the hard cap.
func clampIterations(v int) int {
	if v <= 0 {
		return defaultMaxIterations
	}
	if v > maxMaxIterations {
		return maxMaxIterations
	}
	return v
}

// toolMessage builds a RoleTool result message answering a specific tool call.
func toolMessage(tc llm.ToolCall, data map[string]any) llm.Message {
	return llm.Message{Role: llm.RoleTool, ToolCallID: tc.ID, Name: tc.Name, Content: marshalToolContent(data)}
}

// marshalToolContent renders a tool result as JSON for the model.
func marshalToolContent(data map[string]any) string {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Sprintf("%v", data)
	}
	return string(b)
}

// successOutput builds the agent's final successful output.
func successOutput(answer string, usage llm.Usage, steps []map[string]any) workflow.FunctionOutput {
	return successOutputData(answer, usage, steps)
}

// successOutputData builds the agent's final output; output is the answer text or, for structured
// output (ADR-0030), the validated object.
func successOutputData(output any, usage llm.Usage, steps []map[string]any) workflow.FunctionOutput {
	return workflow.NewFunctionSuccessOutput(map[string]any{
		"output": output,
		"usage": map[string]any{
			"promptTokens":     usage.PromptTokens,
			"completionTokens": usage.CompletionTokens,
			"totalTokens":      usage.TotalTokens,
		},
		"steps": steps,
	})
}

// errorOutput builds a terminal error output.
func errorOutput(msg string) workflow.FunctionOutput {
	return workflow.NewFunctionOutput(workflow.FunctionError, map[string]any{"error": msg})
}
