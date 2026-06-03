package ai

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/open-source-cloud/fuse/pkg/llm"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubProvider returns a canned response (or error) and records the request.
type stubProvider struct {
	name string
	resp llm.ChatResponse
	err  error
	last llm.ChatRequest
}

func (s *stubProvider) Name() string { return s.name }
func (s *stubProvider) Chat(_ context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
	s.last = req
	if s.err != nil {
		return llm.ChatResponse{}, s.err
	}
	return s.resp, nil
}

// runChat invokes the chat function and returns the sync result plus the async
// output delivered via Finish.
func runChat(t *testing.T, reg llm.Registry, input map[string]any) (workflow.FunctionResult, workflow.FunctionOutput) {
	t.Helper()
	fnInput, err := workflow.NewFunctionInputWith(input)
	require.NoError(t, err)

	done := make(chan workflow.FunctionOutput, 1)
	execInfo := workflow.NewExecutionInfo("wf-1", "exec-1", "", fnInput)
	execInfo.Finish = func(out workflow.FunctionOutput) { done <- out }

	res, err := makeChatFunction(reg)(execInfo)
	require.NoError(t, err)

	if !res.Async {
		return res, workflow.FunctionOutput{}
	}
	select {
	case out := <-done:
		return res, out
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for async Finish")
		return res, workflow.FunctionOutput{}
	}
}

func TestChat_SuccessDeliversOutputViaFinish(t *testing.T) {
	prov := &stubProvider{
		name: "stub",
		resp: llm.ChatResponse{
			Message: llm.Message{Role: llm.RoleAssistant, Content: "the answer is 42"},
			Usage:   llm.Usage{PromptTokens: 3, CompletionTokens: 4, TotalTokens: 7},
		},
	}
	reg := llm.NewStaticRegistry(map[string]llm.Provider{"stub": prov}, "stub")

	res, out := runChat(t, reg, map[string]any{
		"input":        "what is the answer?",
		"systemPrompt": "be terse",
		"temperature":  0.5,
	})

	assert.True(t, res.Async)
	assert.Equal(t, workflow.FunctionSuccess, out.Status)
	assert.Equal(t, "the answer is 42", out.Data["output"])

	// system + user messages built in order, temperature forwarded.
	require.Len(t, prov.last.Messages, 2)
	assert.Equal(t, llm.RoleSystem, prov.last.Messages[0].Role)
	assert.Equal(t, llm.RoleUser, prov.last.Messages[1].Role)
	require.NotNil(t, prov.last.Temperature)
	assert.InDelta(t, 0.5, *prov.last.Temperature, 0.0001)
}

func TestChat_MissingInputReturnsSyncError(t *testing.T) {
	reg := llm.NewStaticRegistry(map[string]llm.Provider{"stub": &stubProvider{name: "stub"}}, "stub")
	res, _ := runChat(t, reg, map[string]any{"input": ""})
	assert.False(t, res.Async)
	assert.Equal(t, workflow.FunctionError, res.Output.Status)
}

func TestChat_UnknownProviderDeliveredAsErrorOutput(t *testing.T) {
	// Provider resolution happens inside the async goroutine (it may hit the secret store for
	// per-context keys), so an unknown provider surfaces as an async FunctionError, not a sync one.
	reg := llm.NewStaticRegistry(map[string]llm.Provider{"stub": &stubProvider{name: "stub"}}, "stub")
	res, out := runChat(t, reg, map[string]any{"input": "hi", "provider": "nope"})
	assert.True(t, res.Async)
	assert.Equal(t, workflow.FunctionError, out.Status)
}

func TestChat_ResolvesProviderForExecutionEnvironment(t *testing.T) {
	// A factory registry returns a different provider per environment; the chat function must
	// resolve using execInfo.Environment so per-context keys (ADR-0031) bind to the right env.
	staging := &stubProvider{name: "staging", resp: llm.ChatResponse{Message: llm.Message{Content: "from staging"}}}
	prod := &stubProvider{name: "prod", resp: llm.ChatResponse{Message: llm.Message{Content: "from prod"}}}
	reg := llm.NewRegistry(map[string]llm.ProviderFactory{
		"openai": func(_ context.Context, env string) (llm.Provider, error) {
			if env == "staging" {
				return staging, nil
			}
			return prod, nil
		},
	}, "openai")

	fnInput, err := workflow.NewFunctionInputWith(map[string]any{"input": "hi"})
	require.NoError(t, err)
	done := make(chan workflow.FunctionOutput, 1)
	execInfo := workflow.NewExecutionInfo("wf-1", "exec-1", "staging", fnInput)
	execInfo.Finish = func(out workflow.FunctionOutput) { done <- out }

	res, err := makeChatFunction(reg)(execInfo)
	require.NoError(t, err)
	require.True(t, res.Async)
	select {
	case out := <-done:
		assert.Equal(t, workflow.FunctionSuccess, out.Status)
		assert.Equal(t, "from staging", out.Data["output"])
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for async Finish")
	}
}

func TestChat_ProviderErrorDeliveredAsErrorOutput(t *testing.T) {
	prov := &stubProvider{name: "stub", err: errors.New("boom")}
	reg := llm.NewStaticRegistry(map[string]llm.Provider{"stub": prov}, "stub")
	res, out := runChat(t, reg, map[string]any{"input": "hi"})
	assert.True(t, res.Async)
	assert.Equal(t, workflow.FunctionError, out.Status)
	assert.Contains(t, out.Data["error"], "boom")
}
