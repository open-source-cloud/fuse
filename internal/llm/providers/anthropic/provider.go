// Package anthropic implements the llm.Provider interface against Anthropic's
// native Messages API using anthropic-sdk-go. Unlike the OpenAI-compatible
// providers, Anthropic carries the system prompt outside the message list and
// represents tool calls/results as tool_use / tool_result content blocks.
package anthropic

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/open-source-cloud/fuse/pkg/llm"
)

// defaultMaxTokens is used when a request omits MaxTokens. Anthropic requires an
// explicit maximum, unlike the OpenAI-compatible APIs.
const defaultMaxTokens int64 = 4096

// Config configures an Anthropic provider connection.
type Config struct {
	// Name is the registry key for this provider (i.e. "anthropic").
	Name string
	// APIKey is the Anthropic API key.
	APIKey string
	// BaseURL overrides the API endpoint (for gateways/proxies or tests). Empty
	// uses the SDK default (api.anthropic.com).
	BaseURL string
	// Model is the default model used when a request does not specify one.
	Model string
}

// Provider is an llm.Provider backed by the Anthropic Go SDK.
type Provider struct {
	name         string
	defaultModel string
	client       anthropic.Client
}

// New builds a Provider from cfg.
func New(cfg Config) *Provider {
	opts := make([]option.RequestOption, 0, 2)
	if cfg.APIKey != "" {
		opts = append(opts, option.WithAPIKey(cfg.APIKey))
	}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}
	return &Provider{
		name:         cfg.Name,
		defaultModel: cfg.Model,
		client:       anthropic.NewClient(opts...),
	}
}

// Name returns the provider's registry key.
func (p *Provider) Name() string { return p.name }

// Chat performs a message completion, translating to and from the Anthropic SDK types.
func (p *Provider) Chat(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.defaultModel
	}
	if model == "" {
		return llm.ChatResponse{}, fmt.Errorf("anthropic[%s]: no model specified and no default configured", p.name)
	}

	system, messages := toAnthropicMessages(req.Messages)

	params := anthropic.MessageNewParams{
		Model:     model,
		MaxTokens: resolveMaxTokens(req.MaxTokens),
		Messages:  messages,
		System:    system,
	}
	if len(req.Tools) > 0 {
		params.Tools = toAnthropicTools(req.Tools)
	}
	if req.Temperature != nil {
		params.Temperature = anthropic.Float(float64(*req.Temperature))
	}
	if tc, ok := toAnthropicToolChoice(req.ToolChoice); ok {
		params.ToolChoice = tc
	}

	msg, err := p.client.Messages.New(ctx, params)
	if err != nil {
		return llm.ChatResponse{}, fmt.Errorf("anthropic[%s]: chat completion failed: %w", p.name, err)
	}

	return fromAnthropicMessage(msg), nil
}

// resolveMaxTokens returns the request's max tokens or the default.
func resolveMaxTokens(reqMax *int) int64 {
	if reqMax != nil && *reqMax > 0 {
		return int64(*reqMax)
	}
	return defaultMaxTokens
}

// toAnthropicMessages splits the provider-agnostic messages into Anthropic's
// separate system blocks and the user/assistant/tool message list.
func toAnthropicMessages(messages []llm.Message) ([]anthropic.TextBlockParam, []anthropic.MessageParam) {
	var system []anthropic.TextBlockParam
	out := make([]anthropic.MessageParam, 0, len(messages))
	for _, m := range messages {
		switch m.Role {
		case llm.RoleSystem:
			if m.Content != "" {
				system = append(system, anthropic.TextBlockParam{Text: m.Content})
			}
		case llm.RoleUser:
			out = append(out, anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content)))
		case llm.RoleTool:
			// Anthropic carries tool results in a user-role turn.
			out = append(out, anthropic.NewUserMessage(anthropic.NewToolResultBlock(m.ToolCallID, m.Content, false)))
		case llm.RoleAssistant:
			out = append(out, toAnthropicAssistantMessage(m))
		default:
			out = append(out, anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content)))
		}
	}
	return system, out
}

// toAnthropicAssistantMessage builds an assistant message, carrying any tool calls
// as tool_use content blocks alongside optional text.
func toAnthropicAssistantMessage(m llm.Message) anthropic.MessageParam {
	blocks := make([]anthropic.ContentBlockParamUnion, 0, len(m.ToolCalls)+1)
	if m.Content != "" {
		blocks = append(blocks, anthropic.NewTextBlock(m.Content))
	}
	for _, tc := range m.ToolCalls {
		var input any = tc.Arguments
		if len(tc.Arguments) == 0 {
			input = map[string]any{}
		}
		blocks = append(blocks, anthropic.NewToolUseBlock(tc.ID, input, tc.Name))
	}
	return anthropic.NewAssistantMessage(blocks...)
}

// toAnthropicTools converts tool definitions to Anthropic tool params, extracting
// the JSON-Schema "properties"/"required" from llm.Tool.Parameters.
func toAnthropicTools(tools []llm.Tool) []anthropic.ToolUnionParam {
	out := make([]anthropic.ToolUnionParam, 0, len(tools))
	for _, t := range tools {
		tool := anthropic.ToolParam{
			Name: t.Name,
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: t.Parameters["properties"],
				Required:   toStringSlice(t.Parameters["required"]),
			},
		}
		if t.Description != "" {
			tool.Description = anthropic.String(t.Description)
		}
		out = append(out, anthropic.ToolUnionParam{OfTool: &tool})
	}
	return out
}

// toAnthropicToolChoice maps the agnostic tool-choice string to the SDK union.
func toAnthropicToolChoice(choice string) (anthropic.ToolChoiceUnionParam, bool) {
	switch choice {
	case "auto":
		return anthropic.ToolChoiceUnionParam{OfAuto: &anthropic.ToolChoiceAutoParam{}}, true
	case "required":
		return anthropic.ToolChoiceUnionParam{OfAny: &anthropic.ToolChoiceAnyParam{}}, true
	default:
		return anthropic.ToolChoiceUnionParam{}, false
	}
}

// fromAnthropicMessage converts an SDK response message back to the agnostic type.
func fromAnthropicMessage(msg *anthropic.Message) llm.ChatResponse {
	out := llm.Message{Role: llm.RoleAssistant}
	var content string
	for _, block := range msg.Content {
		switch block.Type {
		case "text":
			content += block.Text
		case "tool_use":
			out.ToolCalls = append(out.ToolCalls, llm.ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: block.Input,
			})
		}
	}
	out.Content = content

	return llm.ChatResponse{
		Message:      out,
		FinishReason: string(msg.StopReason),
		Usage: llm.Usage{
			PromptTokens:     int(msg.Usage.InputTokens),
			CompletionTokens: int(msg.Usage.OutputTokens),
			TotalTokens:      int(msg.Usage.InputTokens + msg.Usage.OutputTokens),
		},
	}
}

// toStringSlice coerces a JSON-Schema "required" value ([]string or []any) to []string.
func toStringSlice(v any) []string {
	switch s := v.(type) {
	case []string:
		return s
	case []any:
		out := make([]string, 0, len(s))
		for _, e := range s {
			if str, ok := e.(string); ok {
				out = append(out, str)
			}
		}
		return out
	default:
		return nil
	}
}
