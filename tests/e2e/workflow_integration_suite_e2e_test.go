//go:build e2e

package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// WorkflowIntegrationSuite runs complex end-to-end scenarios that exercise
// multiple features simultaneously: graph traversal, external HTTP calls,
// loops, and the full foundation feature set.
type WorkflowIntegrationSuite struct {
	suite.Suite
	client       *http.Client
	baseURL      string
	workflowsDir string
}

func TestWorkflowIntegrationSuite(t *testing.T) {
	suite.Run(t, new(WorkflowIntegrationSuite))
}

func (s *WorkflowIntegrationSuite) SetupSuite() {
	s.client, s.baseURL = RequireE2E(s.T())
	s.workflowsDir = WorkflowsDirForTests(s.T())
}

// TestMermaidDAG_TraversesComplexGraph triggers an 11-node DAG workflow
// (all debug/nil nodes) that exercises complex branching and merging paths.
func (s *WorkflowIntegrationSuite) TestMermaidDAG_TraversesComplexGraph() {
	t := s.T()
	wfID := UpsertAndTriggerExampleWorkflow(t, s.client, s.baseURL, s.workflowsDir, "mermaid-test")
	require.NotEmpty(t, wfID)

	resp, err := WaitForWorkflowTerminal(s.client, s.baseURL, wfID, DefaultStatusTimeout)
	require.NoError(t, err, "workflow should reach terminal state")
	assert.Equal(t, "finished", resp.Status,
		"mermaid-test should traverse all DAG paths and finish")
}

// TestMermaidLoop_HandlesBackEdge triggers a DAG with a loop (K -> H back-edge)
// and verifies the workflow engine handles cycles correctly.
func (s *WorkflowIntegrationSuite) TestMermaidLoop_HandlesBackEdge() {
	t := s.T()
	wfID := UpsertAndTriggerExampleWorkflow(t, s.client, s.baseURL, s.workflowsDir, "mermaid-loop-test")
	require.NotEmpty(t, wfID)

	resp, err := WaitForWorkflowTerminal(s.client, s.baseURL, wfID, DefaultStatusTimeout)
	require.NoError(t, err, "workflow should reach terminal state")
	assert.Equal(t, "finished", resp.Status,
		"mermaid-loop-test should handle the loop back-edge and finish")
}

// TestGitHubHTTPRequest_ReachesTerminalState triggers a workflow that makes a
// real HTTP GET to api.github.com. It should finish regardless of whether
// the external call succeeds or fails (conditional routing handles both cases).
func (s *WorkflowIntegrationSuite) TestGitHubHTTPRequest_ReachesTerminalState() {
	t := s.T()
	wfID := UpsertAndTriggerExampleWorkflow(t, s.client, s.baseURL, s.workflowsDir, "github-request-example")
	require.NotEmpty(t, wfID)

	resp, err := WaitForWorkflowTerminal(s.client, s.baseURL, wfID, LongStatusTimeout)
	require.NoError(t, err, "workflow should reach terminal state")
	assert.Contains(t, []string{"finished", "error"}, resp.Status,
		"github-request-example should reach a terminal state (finished or error depending on network)")
}

// TestFullFoundation_ExercisesAllFeatures triggers the comprehensive foundation
// workflow that combines retry with exponential backoff, parallel branches,
// error edges, and a 30s workflow timeout.
func (s *WorkflowIntegrationSuite) TestFullFoundation_ExercisesAllFeatures() {
	t := s.T()
	wfID := UpsertAndTriggerExampleWorkflow(t, s.client, s.baseURL, s.workflowsDir, "full-foundation-test")
	require.NotEmpty(t, wfID)

	resp, err := WaitForWorkflowTerminal(s.client, s.baseURL, wfID, 60*time.Second)
	require.NoError(t, err, "workflow should reach terminal state")
	assert.Equal(t, "finished", resp.Status,
		"full-foundation-test should finish after retries succeed and branches merge")
}
