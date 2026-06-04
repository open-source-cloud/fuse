package ai

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/open-source-cloud/fuse/pkg/llm"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testOutputSchema = []workflow.ParameterSchema{
	{Name: "answer", Type: "string", Required: true},
	{Name: "score", Type: "int", Required: false},
}

func TestValidateAgainstSchema(t *testing.T) {
	t.Parallel()
	assert.NoError(t, validateAgainstSchema(map[string]any{"answer": "hi", "score": float64(5)}, testOutputSchema))
	assert.NoError(t, validateAgainstSchema(map[string]any{"answer": "hi"}, testOutputSchema)) // optional omitted
	assert.Error(t, validateAgainstSchema(map[string]any{"score": float64(5)}, testOutputSchema)) // required missing
	assert.Error(t, validateAgainstSchema(map[string]any{"answer": 1}, testOutputSchema))         // wrong type
}

func TestParseRespond(t *testing.T) {
	t.Parallel()

	obj, err := parseRespond(llm.Message{ToolCalls: []llm.ToolCall{
		{Name: respondToolName, Arguments: json.RawMessage(`{"answer":"hi"}`)},
	}})
	require.NoError(t, err)
	assert.Equal(t, "hi", obj["answer"])

	obj, err = parseRespond(llm.Message{Content: `{"answer":"yo"}`})
	require.NoError(t, err)
	assert.Equal(t, "yo", obj["answer"])

	_, err = parseRespond(llm.Message{Content: "not json"})
	assert.Error(t, err)
}

func TestStructuredOutput_ToolForced(t *testing.T) {
	t.Parallel()
	prov := &stubProvider{
		name: "stub",
		resp: llm.ChatResponse{
			Message: llm.Message{
				Role:      llm.RoleAssistant,
				ToolCalls: []llm.ToolCall{{ID: "1", Name: respondToolName, Arguments: json.RawMessage(`{"answer":"42","score":7}`)}},
			},
			Usage: llm.Usage{PromptTokens: 5, CompletionTokens: 2, TotalTokens: 7},
		},
	}
	obj, u, err := structuredOutput(context.Background(), prov, "m",
		[]llm.Message{{Role: llm.RoleUser, Content: "q"}}, testOutputSchema, NopUsageRecorder{}, ChatFunctionID)
	require.NoError(t, err)
	assert.Equal(t, "42", obj["answer"])
	assert.Equal(t, float64(7), obj["score"])
	assert.Equal(t, 7, u.TotalTokens)
	// The forced request offered exactly the respond tool.
	require.Len(t, prov.last.Tools, 1)
	assert.Equal(t, respondToolName, prov.last.Tools[0].Name)
	assert.Equal(t, "required", prov.last.ToolChoice)
}

func TestChat_StructuredOutputDeliversObject(t *testing.T) {
	prov := &stubProvider{
		name: "stub",
		resp: llm.ChatResponse{
			Message: llm.Message{
				Role:      llm.RoleAssistant,
				ToolCalls: []llm.ToolCall{{ID: "1", Name: respondToolName, Arguments: json.RawMessage(`{"answer":"structured"}`)}},
			},
		},
	}
	reg := llm.NewStaticRegistry(map[string]llm.Provider{"stub": prov}, "stub")

	fnInput, err := workflow.NewFunctionInputWith(map[string]any{
		"input": "hello",
		"outputSchema": []any{
			map[string]any{"name": "answer", "type": "string", "required": true},
		},
	})
	require.NoError(t, err)
	done := make(chan workflow.FunctionOutput, 1)
	execInfo := workflow.NewExecutionInfo("wf-1", "exec-1", "", fnInput)
	execInfo.Finish = func(out workflow.FunctionOutput) { done <- out }

	res, err := makeChatFunction(reg, NopUsageRecorder{})(execInfo)
	require.NoError(t, err)
	require.True(t, res.Async)
	select {
	case out := <-done:
		require.Equal(t, workflow.FunctionSuccess, out.Status)
		obj, ok := out.Data["output"].(map[string]any)
		require.True(t, ok, "output should be a structured object")
		assert.Equal(t, "structured", obj["answer"])
	case <-time.After(2 * time.Second):
		t.Fatal("timed out")
	}
}
