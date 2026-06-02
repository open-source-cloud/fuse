package anthropic_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-source-cloud/fuse/internal/llm/providers/anthropic"
	"github.com/open-source-cloud/fuse/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newStubServer returns an httptest server that records the request body and
// replies with respBody for any /v1/messages request.
func newStubServer(t *testing.T, respBody string, captured *map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if captured != nil {
			_ = json.Unmarshal(body, captured)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, respBody)
	}))
}

func TestProvider_Chat_ParsesTextResponse(t *testing.T) {
	resp := `{
		"id": "msg_1",
		"type": "message",
		"role": "assistant",
		"model": "claude-test",
		"content": [{"type":"text","text":"Hello there"}],
		"stop_reason": "end_turn",
		"usage": {"input_tokens": 5, "output_tokens": 2}
	}`
	var captured map[string]any
	srv := newStubServer(t, resp, &captured)
	defer srv.Close()

	p := anthropic.New(anthropic.Config{Name: "anthropic", APIKey: "test", BaseURL: srv.URL, Model: "claude-test"})
	temp := float32(0.3)
	out, err := p.Chat(context.Background(), llm.ChatRequest{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "be brief"},
			{Role: llm.RoleUser, Content: "hi"},
		},
		Temperature: &temp,
	})

	require.NoError(t, err)
	assert.Equal(t, "Hello there", out.Message.Content)
	assert.Equal(t, llm.RoleAssistant, out.Message.Role)
	assert.Equal(t, "end_turn", out.FinishReason)
	assert.Equal(t, 5, out.Usage.PromptTokens)
	assert.Equal(t, 2, out.Usage.CompletionTokens)
	assert.Equal(t, 7, out.Usage.TotalTokens)

	// Request shape: default model, default max_tokens, system carried separately, temp forwarded.
	assert.Equal(t, "claude-test", captured["model"])
	assert.InDelta(t, 4096, captured["max_tokens"], 0.5, "default max_tokens supplied")
	assert.InDelta(t, 0.3, captured["temperature"], 0.0001)

	system, ok := captured["system"].([]any)
	require.True(t, ok, "system carried as a separate field")
	require.Len(t, system, 1)
	assert.Equal(t, "be brief", system[0].(map[string]any)["text"])

	msgs, ok := captured["messages"].([]any)
	require.True(t, ok)
	require.Len(t, msgs, 1, "system message is not in the messages list")
	assert.Equal(t, "user", msgs[0].(map[string]any)["role"])
}

func TestProvider_Chat_ParsesToolUse(t *testing.T) {
	resp := `{
		"id": "msg_2",
		"type": "message",
		"role": "assistant",
		"model": "claude-test",
		"content": [{"type":"tool_use","id":"toolu_1","name":"http__request","input":{"path":"/x"}}],
		"stop_reason": "tool_use",
		"usage": {"input_tokens": 10, "output_tokens": 3}
	}`
	var captured map[string]any
	srv := newStubServer(t, resp, &captured)
	defer srv.Close()

	p := anthropic.New(anthropic.Config{Name: "anthropic", APIKey: "test", BaseURL: srv.URL})
	out, err := p.Chat(context.Background(), llm.ChatRequest{
		Model:    "claude-test",
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "fetch"}},
		Tools: []llm.Tool{{
			Name:        "http__request",
			Description: "make an http request",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{"path": map[string]any{"type": "string"}},
				"required":   []string{"path"},
			},
		}},
		ToolChoice: "auto",
	})

	require.NoError(t, err)
	require.Len(t, out.Message.ToolCalls, 1)
	tc := out.Message.ToolCalls[0]
	assert.Equal(t, "toolu_1", tc.ID)
	assert.Equal(t, "http__request", tc.Name)
	assert.JSONEq(t, `{"path":"/x"}`, string(tc.Arguments))
	assert.Equal(t, "tool_use", out.FinishReason)

	// The request advertised the tool (with input_schema) and tool_choice=auto.
	tools, ok := captured["tools"].([]any)
	require.True(t, ok)
	require.Len(t, tools, 1)
	tool := tools[0].(map[string]any)
	assert.Equal(t, "http__request", tool["name"])
	assert.Equal(t, "make an http request", tool["description"])
	schema, ok := tool["input_schema"].(map[string]any)
	require.True(t, ok)
	props, ok := schema["properties"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, props, "path")

	choice, ok := captured["tool_choice"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "auto", choice["type"])
}

func TestProvider_Chat_NoModelErrors(t *testing.T) {
	srv := newStubServer(t, `{}`, nil)
	defer srv.Close()

	p := anthropic.New(anthropic.Config{Name: "anthropic", APIKey: "test", BaseURL: srv.URL})
	_, err := p.Chat(context.Background(), llm.ChatRequest{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "hi"}},
	})
	assert.Error(t, err)
}
