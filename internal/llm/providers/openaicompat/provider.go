// Package openaicompat implements the llm.Provider interface against any
// OpenAI-compatible chat completions API. A single implementation, parameterized
// by base URL + API key + default model, serves OpenAI, OpenRouter, Ollama, and
// Gemini's OpenAI-compatible endpoint.
package openaicompat

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/open-source-cloud/fuse/pkg/llm"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"
)

// Config configures an OpenAI-compatible provider connection.
type Config struct {
	// Name is the registry key for this provider (e.g. "openai", "ollama").
	Name string
	// APIKey is the bearer token. May be empty for local backends like Ollama.
	APIKey string
	// BaseURL overrides the API endpoint (e.g. http://localhost:11434/v1 for Ollama).
	BaseURL string
	// Model is the default model used when a request does not specify one.
	Model string
	// Headers are extra HTTP headers sent on every request (e.g. OpenRouter ranking headers).
	Headers map[string]string
}

// Provider is an llm.Provider backed by the OpenAI Go SDK.
type Provider struct {
	name         string
	defaultModel string
	client       openai.Client
}

// New builds a Provider from cfg.
func New(cfg Config) *Provider {
	opts := make([]option.RequestOption, 0, 2+len(cfg.Headers))
	if cfg.APIKey != "" {
		opts = append(opts, option.WithAPIKey(cfg.APIKey))
	}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}
	for k, v := range cfg.Headers {
		opts = append(opts, option.WithHeader(k, v))
	}

	return &Provider{
		name:         cfg.Name,
		defaultModel: cfg.Model,
		client:       openai.NewClient(opts...),
	}
}

// Name returns the provider's registry key.
func (p *Provider) Name() string { return p.name }

// Chat performs a chat completion, translating to and from the OpenAI SDK types.
func (p *Provider) Chat(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.defaultModel
	}
	if model == "" {
		return llm.ChatResponse{}, fmt.Errorf("openaicompat[%s]: no model specified and no default configured", p.name)
	}

	params := openai.ChatCompletionNewParams{
		Model:    model,
		Messages: toOpenAIMessages(req.Messages),
	}
	if len(req.Tools) > 0 {
		params.Tools = toOpenAITools(req.Tools)
	}
	if req.Temperature != nil {
		params.Temperature = openai.Float(float64(*req.Temperature))
	}
	if req.MaxTokens != nil {
		params.MaxCompletionTokens = openai.Int(int64(*req.MaxTokens))
	}
	if req.ToolChoice != "" {
		params.ToolChoice = openai.ChatCompletionToolChoiceOptionUnionParam{
			OfAuto: openai.String(req.ToolChoice),
		}
	}

	completion, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return llm.ChatResponse{}, fmt.Errorf("openaicompat[%s]: chat completion failed: %w", p.name, err)
	}
	if len(completion.Choices) == 0 {
		return llm.ChatResponse{}, fmt.Errorf("openaicompat[%s]: completion returned no choices", p.name)
	}

	choice := completion.Choices[0]
	return llm.ChatResponse{
		Message:      fromOpenAIMessage(choice.Message),
		FinishReason: choice.FinishReason,
		Usage: llm.Usage{
			PromptTokens:     int(completion.Usage.PromptTokens),
			CompletionTokens: int(completion.Usage.CompletionTokens),
			TotalTokens:      int(completion.Usage.TotalTokens),
		},
	}, nil
}

// toOpenAIMessages converts provider-agnostic messages to SDK message params.
func toOpenAIMessages(messages []llm.Message) []openai.ChatCompletionMessageParamUnion {
	out := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))
	for _, m := range messages {
		switch m.Role {
		case llm.RoleSystem:
			out = append(out, openai.SystemMessage(m.Content))
		case llm.RoleUser:
			out = append(out, openai.UserMessage(m.Content))
		case llm.RoleTool:
			out = append(out, openai.ToolMessage(m.Content, m.ToolCallID))
		case llm.RoleAssistant:
			out = append(out, toOpenAIAssistantMessage(m))
		default:
			out = append(out, openai.UserMessage(m.Content))
		}
	}
	return out
}

// toOpenAIAssistantMessage builds an assistant message param, carrying any tool calls.
func toOpenAIAssistantMessage(m llm.Message) openai.ChatCompletionMessageParamUnion {
	if len(m.ToolCalls) == 0 {
		return openai.AssistantMessage(m.Content)
	}

	asst := openai.ChatCompletionAssistantMessageParam{
		ToolCalls: make([]openai.ChatCompletionMessageToolCallUnionParam, 0, len(m.ToolCalls)),
	}
	if m.Content != "" {
		asst.Content.OfString = openai.String(m.Content)
	}
	for _, tc := range m.ToolCalls {
		asst.ToolCalls = append(asst.ToolCalls, openai.ChatCompletionMessageToolCallUnionParam{
			OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
				ID: tc.ID,
				Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
					Name:      tc.Name,
					Arguments: string(tc.Arguments),
				},
			},
		})
	}
	return openai.ChatCompletionMessageParamUnion{OfAssistant: &asst}
}

// toOpenAITools converts tool definitions to SDK tool params.
func toOpenAITools(tools []llm.Tool) []openai.ChatCompletionToolUnionParam {
	out := make([]openai.ChatCompletionToolUnionParam, 0, len(tools))
	for _, t := range tools {
		fn := shared.FunctionDefinitionParam{
			Name:       t.Name,
			Parameters: shared.FunctionParameters(t.Parameters),
		}
		if t.Description != "" {
			fn.Description = openai.String(t.Description)
		}
		out = append(out, openai.ChatCompletionFunctionTool(fn))
	}
	return out
}

// fromOpenAIMessage converts an SDK response message back to the agnostic type.
func fromOpenAIMessage(m openai.ChatCompletionMessage) llm.Message {
	msg := llm.Message{
		Role:    llm.RoleAssistant,
		Content: m.Content,
	}
	if len(m.ToolCalls) > 0 {
		msg.ToolCalls = make([]llm.ToolCall, 0, len(m.ToolCalls))
		for _, tc := range m.ToolCalls {
			if tc.Type != "function" {
				continue
			}
			msg.ToolCalls = append(msg.ToolCalls, llm.ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: json.RawMessage(tc.Function.Arguments),
			})
		}
	}
	return msg
}
