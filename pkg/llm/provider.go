// Package llm defines the provider-agnostic interface and value types used to
// talk to Large Language Model backends (OpenAI, OpenRouter, Ollama, Gemini,
// Anthropic, ...). The types here are intentionally SDK-agnostic so a single
// agent reasoning loop can drive every provider, and so a provider backed by a
// different SDK can be swapped in without touching callers.
package llm

import (
	"context"
	"encoding/json"
)

// Role identifies the author of a chat Message.
type Role string

const (
	// RoleSystem is the system / developer instruction turn.
	RoleSystem Role = "system"
	// RoleUser is an end-user turn.
	RoleUser Role = "user"
	// RoleAssistant is a model turn (may carry ToolCalls).
	RoleAssistant Role = "assistant"
	// RoleTool is a tool-result turn fed back to the model.
	RoleTool Role = "tool"
)

// Message is a single turn in a chat conversation.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
	// ToolCalls is set on assistant turns that request one or more tool invocations.
	ToolCalls []ToolCall `json:"toolCalls,omitempty"`
	// ToolCallID identifies which ToolCall a RoleTool result answers.
	ToolCallID string `json:"toolCallId,omitempty"`
	// Name is the tool name on a RoleTool result turn.
	Name string `json:"name,omitempty"`
}

// ToolCall is a model request to invoke a tool. Arguments is the raw JSON
// object the model produced for the tool's parameters.
type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// Tool is a tool/function definition advertised to the model. Parameters is a
// JSON Schema object describing the tool's accepted arguments.
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// ChatRequest is a single provider-agnostic chat completion request.
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Tools       []Tool    `json:"tools,omitempty"`
	Temperature *float32  `json:"temperature,omitempty"`
	MaxTokens   *int      `json:"maxTokens,omitempty"`
	// ToolChoice is "auto", "none", or "required" (empty defaults to the provider's behavior).
	ToolChoice string `json:"toolChoice,omitempty"`
}

// ChatResponse is the result of a chat completion.
type ChatResponse struct {
	Message      Message `json:"message"`
	FinishReason string  `json:"finishReason"`
	Usage        Usage   `json:"usage"`
}

// Usage reports token accounting for a completion.
type Usage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

// Provider is the thin seam every LLM backend implements. A single Chat method
// covers tool-calling chat completion across all providers.
type Provider interface {
	// Name returns the provider's registry key (e.g. "openai", "ollama").
	Name() string
	// Chat performs a (blocking) chat completion.
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}

// StreamChunk is one incremental piece of a streamed completion (Phase C).
type StreamChunk struct {
	// ContentDelta is the incremental assistant text for this chunk.
	ContentDelta string `json:"contentDelta"`
	// Done is true on the final chunk; Response carries the aggregated result.
	Done     bool          `json:"done"`
	Response *ChatResponse `json:"response,omitempty"`
	Err      error         `json:"-"`
}

// StreamingProvider is an optional capability. Detect support with a type
// assertion on a Provider value.
type StreamingProvider interface {
	Provider
	// ChatStream performs a streaming chat completion.
	ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error)
}
