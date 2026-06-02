package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldSkipExampleWorkflow_CI(t *testing.T) {
	t.Parallel()
	assert.True(t, shouldSkipExampleWorkflow("github-request-example.json", []byte(`{"id":"x"}`)))
	assert.False(t, shouldSkipExampleWorkflow("smallest-test.json", []byte(`{"id":"smallest"}`)))
	assert.True(t, shouldSkipExampleWorkflow("sleep-test.json", []byte(`"function": "fuse/pkg/logic/timer/foo"`)))
	assert.True(t, shouldSkipExampleWorkflow("ai-agent-example.json", []byte(`"function": "fuse/pkg/ai/agent"`)))
	assert.True(t, shouldSkipExampleWorkflow("ai-chat-example.json", []byte(`"function": "fuse/pkg/ai/chat"`)))
}
