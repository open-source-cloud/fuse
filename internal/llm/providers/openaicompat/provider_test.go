package openaicompat_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-source-cloud/fuse/internal/llm/providers/openaicompat"
	"github.com/open-source-cloud/fuse/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newStubServer returns an httptest server that records the last request body and
// replies with respBody for any chat completion request.
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
		"id": "chatcmpl-1",
		"object": "chat.completion",
		"choices": [{"index":0,"message":{"role":"assistant","content":"Hello there"},"finish_reason":"stop"}],
		"usage": {"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}
	}`
	var captured map[string]any
	srv := newStubServer(t, resp, &captured)
	defer srv.Close()

	p := openaicompat.New(openaicompat.Config{Name: "stub", BaseURL: srv.URL, Model: "test-model"})
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
	assert.Equal(t, "stop", out.FinishReason)
	assert.Equal(t, 7, out.Usage.TotalTokens)

	// The request used the configured default model and carried both messages + temperature.
	assert.Equal(t, "test-model", captured["model"])
	assert.InDelta(t, 0.3, captured["temperature"], 0.0001)
	msgs, ok := captured["messages"].([]any)
	require.True(t, ok)
	assert.Len(t, msgs, 2)
}

func TestProvider_Chat_ParsesToolCalls(t *testing.T) {
	resp := `{
		"id": "chatcmpl-2",
		"object": "chat.completion",
		"choices": [{"index":0,"finish_reason":"tool_calls","message":{"role":"assistant","content":"",
			"tool_calls":[{"id":"call_1","type":"function","function":{"name":"http__request","arguments":"{\"path\":\"/x\"}"}}]}}],
		"usage": {"prompt_tokens":10,"completion_tokens":3,"total_tokens":13}
	}`
	var captured map[string]any
	srv := newStubServer(t, resp, &captured)
	defer srv.Close()

	p := openaicompat.New(openaicompat.Config{Name: "stub", BaseURL: srv.URL})
	out, err := p.Chat(context.Background(), llm.ChatRequest{
		Model:    "test-model",
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "fetch"}},
		Tools: []llm.Tool{{
			Name:        "http__request",
			Description: "make an http request",
			Parameters:  map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}}},
		}},
		ToolChoice: "auto",
	})

	require.NoError(t, err)
	require.Len(t, out.Message.ToolCalls, 1)
	tc := out.Message.ToolCalls[0]
	assert.Equal(t, "call_1", tc.ID)
	assert.Equal(t, "http__request", tc.Name)
	assert.JSONEq(t, `{"path":"/x"}`, string(tc.Arguments))

	// The request advertised the tool and tool_choice.
	tools, ok := captured["tools"].([]any)
	require.True(t, ok)
	assert.Len(t, tools, 1)
	assert.Equal(t, "auto", captured["tool_choice"])
}

func TestProvider_Chat_NoModelErrors(t *testing.T) {
	srv := newStubServer(t, `{}`, nil)
	defer srv.Close()

	p := openaicompat.New(openaicompat.Config{Name: "stub", BaseURL: srv.URL})
	_, err := p.Chat(context.Background(), llm.ChatRequest{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "hi"}},
	})
	assert.Error(t, err)
}
