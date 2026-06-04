package ai

import (
	"context"
	"errors"
	"time"

	"github.com/open-source-cloud/fuse/internal/packages/transport"
	"github.com/open-source-cloud/fuse/pkg/llm"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/rs/zerolog/log"
)

// ChatFunctionID is the id of the chat function.
const ChatFunctionID = "chat"

// chatTimeout bounds a single chat completion call.
const chatTimeout = 2 * time.Minute

// ErrInputRequired is returned when the required input message is missing.
var ErrInputRequired = errors.New("ai/chat: input is required")

// ChatFunctionMetadata returns the metadata for the chat function.
func ChatFunctionMetadata() workflow.FunctionMetadata {
	return workflow.FunctionMetadata{
		Transport: transport.Internal,
		Input: workflow.InputMetadata{
			CustomParameters: false,
			Parameters: []workflow.ParameterSchema{
				{
					Name:        "input",
					Type:        "string",
					Required:    true,
					Description: "The user message / prompt sent to the model",
				},
				{
					Name:        "provider",
					Type:        "string",
					Required:    false,
					Description: "Provider registry key (e.g. openai, ollama). Defaults to the configured default provider",
				},
				{
					Name:        "model",
					Type:        "string",
					Required:    false,
					Description: "Model id. Defaults to the provider's configured default model",
				},
				{
					Name:        "systemPrompt",
					Type:        "string",
					Required:    false,
					Description: "Optional system instruction prepended to the conversation",
				},
				{
					Name:        "temperature",
					Type:        "float",
					Required:    false,
					Description: "Sampling temperature; if omitted the provider default is used",
				},
			},
			Edges: workflow.InputEdgeMetadata{
				Count:      0,
				Parameters: make([]workflow.ParameterSchema, 0),
			},
		},
		Output: workflow.OutputMetadata{
			Parameters: []workflow.ParameterSchema{
				{
					Name:        "output",
					Type:        "string",
					Required:    true,
					Description: "The model's text response",
				},
				{
					Name:        "usage",
					Type:        "map",
					Required:    false,
					Description: "Token usage: promptTokens, completionTokens, totalTokens",
				},
			},
			Edges: make([]workflow.OutputEdgeMetadata, 0),
		},
	}
}

// makeChatFunction builds the ai/chat function, closing over the provider registry and the
// usage recorder (ADR-0029).
func makeChatFunction(providers llm.Registry, usage UsageRecorder) workflow.Function {
	return func(execInfo *workflow.ExecutionInfo) (workflow.FunctionResult, error) {
		input := execInfo.Input

		userInput := input.GetStr("input")
		if userInput == "" {
			return workflow.NewFunctionResultError(ErrInputRequired)
		}

		providerName := input.GetStr("provider")

		messages := make([]llm.Message, 0, 2)
		if systemPrompt := input.GetStr("systemPrompt"); systemPrompt != "" {
			messages = append(messages, llm.Message{Role: llm.RoleSystem, Content: systemPrompt})
		}
		messages = append(messages, llm.Message{Role: llm.RoleUser, Content: userInput})

		req := llm.ChatRequest{
			Model:       input.GetStr("model"),
			Messages:    messages,
			Temperature: optionalTemperature(input),
		}

		// Provider resolution and the completion run in their own goroutine and report back via
		// Finish so the WorkflowFunc pool worker is freed immediately (mirrors logic/timer).
		// Resolution is in here too because per-context provider keys (ADR-0031) may hit the
		// secret store, which is I/O we must keep off the pool worker.
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), chatTimeout)
			defer cancel()

			provider, err := resolveProvider(ctx, providers, execInfo.Environment, providerName)
			if err != nil {
				log.Error().Err(err).Str("provider", providerName).Msg("ai/chat provider resolution failed")
				execInfo.Finish(workflow.NewFunctionOutput(workflow.FunctionError, map[string]any{"error": err.Error()}))
				return
			}

			resp, err := provider.Chat(ctx, req)
			if err != nil {
				usage.RecordCall(ChatFunctionID, provider.Name(), req.Model, "error")
				log.Error().Err(err).Str("provider", provider.Name()).Msg("ai/chat completion failed")
				execInfo.Finish(workflow.NewFunctionOutput(workflow.FunctionError, map[string]any{"error": err.Error()}))
				return
			}
			usage.RecordCall(ChatFunctionID, provider.Name(), req.Model, "success")
			usage.RecordUsage(ChatFunctionID, provider.Name(), req.Model, resp.Usage)

			execInfo.Finish(workflow.NewFunctionSuccessOutput(map[string]any{
				"output": resp.Message.Content,
				"usage": map[string]any{
					"promptTokens":     resp.Usage.PromptTokens,
					"completionTokens": resp.Usage.CompletionTokens,
					"totalTokens":      resp.Usage.TotalTokens,
				},
			}))
		}()

		return workflow.NewFunctionResultAsync(), nil
	}
}

// resolveProvider returns the named provider, or the registry default when name is empty, built
// for the given environment so per-context keys (ADR-0031) resolve against the running workflow.
func resolveProvider(ctx context.Context, providers llm.Registry, environment, name string) (llm.Provider, error) {
	if name != "" {
		return providers.Get(ctx, environment, name)
	}
	return providers.Default(ctx, environment)
}

// optionalTemperature extracts the temperature input if present and numeric.
func optionalTemperature(input *workflow.FunctionInput) *float32 {
	raw := input.Get("temperature")
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case float64:
		t := float32(v)
		return &t
	case float32:
		return &v
	case int:
		t := float32(v)
		return &t
	default:
		return nil
	}
}
