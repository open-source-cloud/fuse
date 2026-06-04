package ai

import (
	"sync"
	"testing"
	"time"

	"github.com/open-source-cloud/fuse/pkg/llm"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeUsageRecorder captures RecordUsage / RecordCall invocations.
type fakeUsageRecorder struct {
	mu     sync.Mutex
	usage  []llm.Usage
	status []string
}

func (f *fakeUsageRecorder) RecordUsage(_, _, _ string, u llm.Usage) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.usage = append(f.usage, u)
}

func (f *fakeUsageRecorder) RecordCall(_, _, _, status string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status = append(f.status, status)
}

func (f *fakeUsageRecorder) snapshot() ([]llm.Usage, []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]llm.Usage(nil), f.usage...), append([]string(nil), f.status...)
}

func TestChat_RecordsUsageMetrics(t *testing.T) {
	prov := &stubProvider{
		name: "stub",
		resp: llm.ChatResponse{
			Message: llm.Message{Role: llm.RoleAssistant, Content: "hi"},
			Usage:   llm.Usage{PromptTokens: 3, CompletionTokens: 4, TotalTokens: 7},
		},
	}
	reg := llm.NewStaticRegistry(map[string]llm.Provider{"stub": prov}, "stub")
	rec := &fakeUsageRecorder{}

	fnInput, err := workflow.NewFunctionInputWith(map[string]any{"input": "hello"})
	require.NoError(t, err)
	done := make(chan workflow.FunctionOutput, 1)
	execInfo := workflow.NewExecutionInfo("wf-1", "exec-1", "", fnInput)
	execInfo.Finish = func(out workflow.FunctionOutput) { done <- out }

	res, err := makeChatFunction(reg, rec)(execInfo)
	require.NoError(t, err)
	require.True(t, res.Async)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out")
	}

	usage, status := rec.snapshot()
	require.Len(t, usage, 1)
	assert.Equal(t, 3, usage[0].PromptTokens)
	assert.Equal(t, 4, usage[0].CompletionTokens)
	assert.Equal(t, []string{"success"}, status)

	// The no-op recorder must satisfy the interface and not panic.
	var nop UsageRecorder = NopUsageRecorder{}
	nop.RecordUsage(ChatFunctionID, "stub", "m", llm.Usage{PromptTokens: 1})
	nop.RecordCall(ChatFunctionID, "stub", "m", "success")
}
