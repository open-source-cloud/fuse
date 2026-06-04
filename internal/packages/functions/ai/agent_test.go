package ai

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/open-source-cloud/fuse/pkg/llm"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- test doubles -----------------------------------------------------------

// scriptedProvider returns a sequence of canned responses and records requests.
type scriptedProvider struct {
	name      string
	responses []llm.ChatResponse
	err       error
	errOnCall int // call index (0-based) on which to return err
	repeat    bool
	requests  []llm.ChatRequest
	calls     int
}

func (s *scriptedProvider) Name() string { return s.name }

func (s *scriptedProvider) Chat(_ context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
	s.requests = append(s.requests, req)
	i := s.calls
	s.calls++
	if s.err != nil && i == s.errOnCall {
		return llm.ChatResponse{}, s.err
	}
	if i < len(s.responses) {
		return s.responses[i], nil
	}
	if s.repeat && len(s.responses) > 0 {
		return s.responses[len(s.responses)-1], nil
	}
	return finalAnswer("fallback"), nil
}

type fakeToolRegistry struct {
	descriptors []ToolDescriptor
	invoke      func(string, *workflow.ExecutionInfo) (workflow.FunctionResult, error)
	invoked     []string
}

func (f *fakeToolRegistry) ListTools() []ToolDescriptor { return f.descriptors }

func (f *fakeToolRegistry) InvokeTool(id string, e *workflow.ExecutionInfo) (workflow.FunctionResult, error) {
	f.invoked = append(f.invoked, id)
	if f.invoke != nil {
		return f.invoke(id, e)
	}
	return workflow.NewFunctionResultSuccessWith(map[string]any{"ok": true}), nil
}

// --- helpers ----------------------------------------------------------------

func finalAnswer(text string) llm.ChatResponse {
	return llm.ChatResponse{
		Message:      llm.Message{Role: llm.RoleAssistant, Content: text},
		FinishReason: "stop",
		Usage:        llm.Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2},
	}
}

func toolCallResponse(callID, mangledName, argsJSON string) llm.ChatResponse {
	return llm.ChatResponse{
		Message: llm.Message{
			Role:      llm.RoleAssistant,
			ToolCalls: []llm.ToolCall{{ID: callID, Name: mangledName, Arguments: json.RawMessage(argsJSON)}},
		},
		FinishReason: "tool_calls",
		Usage:        llm.Usage{PromptTokens: 2, CompletionTokens: 2, TotalTokens: 4},
	}
}

var sumDescriptor = ToolDescriptor{
	FunctionID:  "fuse/pkg/logic/sum",
	MangledName: "fuse__pkg__logic__sum",
	Description: "sum",
	Parameters:  map[string]any{"type": "object", "properties": map[string]any{"values": map[string]any{"type": "array"}}},
}

var randDescriptor = ToolDescriptor{
	FunctionID:  "fuse/pkg/logic/rand",
	MangledName: "fuse__pkg__logic__rand",
	Description: "rand",
	Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
}

func registryWith(prov llm.Provider) llm.Registry {
	return llm.NewStaticRegistry(map[string]llm.Provider{prov.Name(): prov}, prov.Name())
}

func runAgent(t *testing.T, providers llm.Registry, tools ToolRegistry, input map[string]any) (workflow.FunctionResult, workflow.FunctionOutput) {
	t.Helper()
	fnInput, err := workflow.NewFunctionInputWith(input)
	require.NoError(t, err)

	done := make(chan workflow.FunctionOutput, 1)
	execInfo := workflow.NewExecutionInfo("wf-1", workflow.NewExecID(1), "", fnInput)
	execInfo.Finish = func(out workflow.FunctionOutput) { done <- out }

	res, err := makeAgentFunction(providers, tools, NopUsageRecorder{})(execInfo)
	require.NoError(t, err)
	if !res.Async {
		return res, workflow.FunctionOutput{}
	}
	select {
	case out := <-done:
		return res, out
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for async Finish")
		return res, workflow.FunctionOutput{}
	}
}

func findToolMessage(msgs []llm.Message) (llm.Message, bool) {
	for _, m := range msgs {
		if m.Role == llm.RoleTool {
			return m, true
		}
	}
	return llm.Message{}, false
}

// --- tests ------------------------------------------------------------------

func TestAgent_HappyPathSingleToolCall(t *testing.T) {
	prov := &scriptedProvider{
		name: "stub",
		responses: []llm.ChatResponse{
			toolCallResponse("call-1", "fuse__pkg__logic__sum", `{"values":[2,3]}`),
			finalAnswer("the sum is 5"),
		},
	}
	tools := &fakeToolRegistry{
		descriptors: []ToolDescriptor{sumDescriptor},
		invoke: func(_ string, _ *workflow.ExecutionInfo) (workflow.FunctionResult, error) {
			return workflow.NewFunctionResultSuccessWith(map[string]any{"sum": 5}), nil
		},
	}

	res, out := runAgent(t, registryWith(prov), tools, map[string]any{"input": "add 2 and 3"})

	require.True(t, res.Async)
	assert.Equal(t, workflow.FunctionSuccess, out.Status)
	assert.Equal(t, "the sum is 5", out.Data["output"])

	// the sum tool was actually invoked
	assert.Equal(t, []string{"fuse/pkg/logic/sum"}, tools.invoked)

	// the first request advertised the sum tool
	require.Len(t, prov.requests, 2)
	require.Len(t, prov.requests[0].Tools, 1)
	assert.Equal(t, "fuse__pkg__logic__sum", prov.requests[0].Tools[0].Name)
	assert.Equal(t, "auto", prov.requests[0].ToolChoice)

	// the second request fed the tool result back, threaded by ToolCallID
	toolMsg, ok := findToolMessage(prov.requests[1].Messages)
	require.True(t, ok, "second request should include a tool-result message")
	assert.Equal(t, "call-1", toolMsg.ToolCallID)
	assert.Contains(t, toolMsg.Content, "5")

	// usage aggregated across both calls
	usage, ok := out.Data["usage"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 3, usage["promptTokens"])
	assert.Equal(t, 6, usage["totalTokens"])

	// trace recorded the call
	steps, ok := out.Data["steps"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, steps, 1)
	assert.Equal(t, "fuse/pkg/logic/sum", steps[0]["tool"])
	assert.Equal(t, map[string]any{"sum": 5}, steps[0]["result"])
}

func TestAgent_DirectAnswerNoTools(t *testing.T) {
	prov := &scriptedProvider{name: "stub", responses: []llm.ChatResponse{finalAnswer("hello there")}}
	tools := &fakeToolRegistry{}

	_, out := runAgent(t, registryWith(prov), tools, map[string]any{"input": "say hello"})

	assert.Equal(t, workflow.FunctionSuccess, out.Status)
	assert.Equal(t, "hello there", out.Data["output"])
	assert.Empty(t, tools.invoked)
	steps, ok := out.Data["steps"].([]map[string]any)
	require.True(t, ok)
	assert.Empty(t, steps)
}

func TestAgent_MissingInputReturnsSyncError(t *testing.T) {
	prov := &scriptedProvider{name: "stub"}
	res, _ := runAgent(t, registryWith(prov), &fakeToolRegistry{}, map[string]any{"input": ""})
	assert.False(t, res.Async)
	assert.Equal(t, workflow.FunctionError, res.Output.Status)
}

func TestAgent_UnknownProviderDeliveredAsErrorOutput(t *testing.T) {
	// Provider resolution happens inside the async goroutine (it may hit the secret store for
	// per-context keys), so an unknown provider surfaces as an async FunctionError, not a sync one.
	prov := &scriptedProvider{name: "stub"}
	res, out := runAgent(t, registryWith(prov), &fakeToolRegistry{}, map[string]any{"input": "hi", "provider": "nope"})
	assert.True(t, res.Async)
	assert.Equal(t, workflow.FunctionError, out.Status)
}

func TestAgent_UnknownToolFedBackAndContinues(t *testing.T) {
	prov := &scriptedProvider{
		name: "stub",
		responses: []llm.ChatResponse{
			toolCallResponse("c1", "bogus__tool", `{}`),
			finalAnswer("recovered"),
		},
	}
	tools := &fakeToolRegistry{descriptors: []ToolDescriptor{sumDescriptor}}

	_, out := runAgent(t, registryWith(prov), tools, map[string]any{"input": "use a tool"})

	assert.Equal(t, workflow.FunctionSuccess, out.Status)
	assert.Equal(t, "recovered", out.Data["output"])
	assert.Empty(t, tools.invoked, "an unknown tool is never invoked")

	steps := out.Data["steps"].([]map[string]any)
	require.Len(t, steps, 1)
	assert.Equal(t, "bogus__tool", steps[0]["tool"])
	assert.Contains(t, steps[0]["error"], "unknown or disallowed tool")

	toolMsg, ok := findToolMessage(prov.requests[1].Messages)
	require.True(t, ok)
	assert.Contains(t, toolMsg.Content, "error")
}

func TestAgent_MaxIterationsReached(t *testing.T) {
	prov := &scriptedProvider{
		name:      "stub",
		responses: []llm.ChatResponse{toolCallResponse("c1", "fuse__pkg__logic__sum", `{}`)},
		repeat:    true,
	}
	tools := &fakeToolRegistry{
		descriptors: []ToolDescriptor{sumDescriptor},
		invoke: func(_ string, _ *workflow.ExecutionInfo) (workflow.FunctionResult, error) {
			return workflow.NewFunctionResultSuccessWith(map[string]any{"sum": 0}), nil
		},
	}

	_, out := runAgent(t, registryWith(prov), tools, map[string]any{"input": "loop forever", "maxIterations": 3})

	assert.Equal(t, workflow.FunctionError, out.Status)
	assert.Contains(t, out.Data["error"], "max iterations")
	assert.Equal(t, 3, prov.calls, "should stop exactly at the iteration limit")
}

func TestAgent_ToolErrorSurfacedAndContinues(t *testing.T) {
	prov := &scriptedProvider{
		name: "stub",
		responses: []llm.ChatResponse{
			toolCallResponse("c1", "fuse__pkg__logic__sum", `{}`),
			finalAnswer("ok"),
		},
	}
	tools := &fakeToolRegistry{
		descriptors: []ToolDescriptor{sumDescriptor},
		invoke: func(_ string, _ *workflow.ExecutionInfo) (workflow.FunctionResult, error) {
			return workflow.NewFunctionResult(workflow.FunctionError, map[string]any{"error": "boom"}), nil
		},
	}

	_, out := runAgent(t, registryWith(prov), tools, map[string]any{"input": "use tool"})

	assert.Equal(t, workflow.FunctionSuccess, out.Status)
	steps := out.Data["steps"].([]map[string]any)
	require.Len(t, steps, 1)
	assert.Contains(t, steps[0]["error"], "tool returned an error")
}

func TestAgent_AsyncToolUnsupported(t *testing.T) {
	prov := &scriptedProvider{
		name: "stub",
		responses: []llm.ChatResponse{
			toolCallResponse("c1", "fuse__pkg__logic__sum", `{}`),
			finalAnswer("done"),
		},
	}
	tools := &fakeToolRegistry{
		descriptors: []ToolDescriptor{sumDescriptor},
		invoke: func(_ string, _ *workflow.ExecutionInfo) (workflow.FunctionResult, error) {
			return workflow.NewFunctionResultAsync(), nil
		},
	}

	_, out := runAgent(t, registryWith(prov), tools, map[string]any{"input": "use async tool"})

	assert.Equal(t, workflow.FunctionSuccess, out.Status)
	steps := out.Data["steps"].([]map[string]any)
	require.Len(t, steps, 1)
	assert.Contains(t, steps[0]["error"], "asynchronous")
}

func TestAgent_AllowedToolsFilter(t *testing.T) {
	prov := &scriptedProvider{name: "stub", responses: []llm.ChatResponse{finalAnswer("ok")}}
	tools := &fakeToolRegistry{descriptors: []ToolDescriptor{sumDescriptor, randDescriptor}}

	_, out := runAgent(t, registryWith(prov), tools, map[string]any{
		"input":        "do it",
		"allowedTools": []any{"fuse/pkg/logic/sum"},
	})

	assert.Equal(t, workflow.FunctionSuccess, out.Status)
	require.Len(t, prov.requests, 1)
	require.Len(t, prov.requests[0].Tools, 1, "only the allowlisted tool should be offered")
	assert.Equal(t, "fuse__pkg__logic__sum", prov.requests[0].Tools[0].Name)
}

func TestAgent_ProviderErrorDeliveredAsErrorOutput(t *testing.T) {
	prov := &scriptedProvider{name: "stub", err: errors.New("upstream boom"), errOnCall: 0}
	_, out := runAgent(t, registryWith(prov), &fakeToolRegistry{}, map[string]any{"input": "hi"})
	assert.Equal(t, workflow.FunctionError, out.Status)
	assert.Contains(t, out.Data["error"], "boom")
}

func TestClampIterations(t *testing.T) {
	t.Parallel()
	assert.Equal(t, defaultMaxIterations, clampIterations(0))
	assert.Equal(t, defaultMaxIterations, clampIterations(-5))
	assert.Equal(t, 7, clampIterations(7))
	assert.Equal(t, maxMaxIterations, clampIterations(1000))
}

func TestAgentFunctionMetadata_Shape(t *testing.T) {
	t.Parallel()
	meta := AgentFunctionMetadata()
	assert.False(t, meta.Input.CustomParameters)

	names := make(map[string]bool)
	for _, p := range meta.Input.Parameters {
		names[p.Name] = p.Required
	}
	require.Contains(t, names, "input")
	assert.True(t, names["input"], "input is required")
	assert.Contains(t, names, "maxIterations")
	assert.Contains(t, names, "allowedTools")
}
