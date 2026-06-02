//go:build e2e

package e2e

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// aiStatusTimeout is generous on purpose: a small local model on a CI runner is slow, and the
// ai/agent node itself allows up to ~5 minutes per interaction.
const aiStatusTimeout = 5 * time.Minute

// TestAIAgentExample_E2E runs the ai/agent example workflow against a live, Ollama-backed server
// and asserts it completes successfully.
//
// It is gated by AI_E2E=true so the standard e2e job (which has no LLM provider) skips it; the
// dedicated ai-e2e workflow sets AI_E2E and the LLM_OLLAMA_* env on the server. The e2e overlay
// variant (examples/workflows/e2e/ai-agent-example.json) keeps the task small and bounds
// maxIterations for speed.
//
// Reaching the "finished" terminal state is a strong end-to-end signal: the agent reached the live
// model, advertised its tools, ran the reasoning loop, invoked any requested tool in-process, fed
// the result back, and delivered a final answer via Finish. A regression in the provider, the agent
// loop, or the tool plumbing would surface as "error" instead.
func TestAIAgentExample_E2E(t *testing.T) {
	if os.Getenv("AI_E2E") != "true" {
		t.Skip("AI e2e disabled; set AI_E2E=true with a running Ollama-backed server to enable")
	}

	client, base := RequireE2E(t)
	workflowsDir := WorkflowsDirForTests(t)

	wfID := UpsertAndTriggerExampleWorkflow(t, client, base, workflowsDir, "ai-agent-example")

	status, err := WaitForWorkflowTerminal(client, base, wfID, aiStatusTimeout)
	require.NoError(t, err, errMsgWorkflowShouldReachTerminal)

	// Log the full workflow record (agent output / steps) for visibility; best-effort.
	if _, body, gerr := GET(client, base+"/v1/workflows/"+wfID); gerr == nil {
		t.Logf("ai-agent workflow %s result: %s", wfID, string(body))
	}

	require.Equalf(t, "finished", status.Status,
		"ai/agent workflow %s should finish, not error — the agent must reach a final answer against the live model", wfID)
}
