//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// WorkflowResilienceSuite verifies error handling, retry mechanisms,
// and timeout recovery in workflow execution.
type WorkflowResilienceSuite struct {
	suite.Suite
	client       *http.Client
	baseURL      string
	workflowsDir string
}

func TestWorkflowResilienceSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(WorkflowResilienceSuite))
}

func (s *WorkflowResilienceSuite) SetupSuite() {
	s.client, s.baseURL = RequireE2E(s.T())
	s.workflowsDir = WorkflowsDirForTests(s.T())
	workflows := []string{
		"error-edge-test", "retry-test", "parallel-retry-test", "timeout-test",
	}
	for _, id := range workflows {
		UpsertSchema(s.T(), s.client, s.baseURL, s.workflowsDir, id)
	}
}

// TestErrorEdge_FollowsRecoveryPath triggers a workflow where a node always fails
// (maxAttempts=0) and verifies the error edge routes to the recovery node,
// completing the workflow successfully.
func (s *WorkflowResilienceSuite) TestErrorEdge_FollowsRecoveryPath() {
	t := s.T()
	wfID := TriggerExampleWorkflow(t, s.client, s.baseURL, "error-edge-test")
	require.NotEmpty(t, wfID)

	resp, err := WaitForWorkflowTerminal(s.client, s.baseURL, wfID, FastStatusTimeout)
	require.NoError(t, err, errMsgWorkflowShouldReachTerminal)
	assert.Equal(t, wfID, resp.WorkflowID)
	assert.Equal(t, "finished", resp.Status,
		"error-edge-test should finish via the recovery path, not stay in error")
}

// TestRetry_CompletesAfterTransientFailures triggers a workflow where a node fails
// twice then succeeds on the third attempt (maxAttempts=3, failCount=2).
func (s *WorkflowResilienceSuite) TestRetry_CompletesAfterTransientFailures() {
	t := s.T()
	wfID := TriggerExampleWorkflow(t, s.client, s.baseURL, "retry-test")
	require.NotEmpty(t, wfID)

	resp, err := WaitForWorkflowTerminal(s.client, s.baseURL, wfID, FastStatusTimeout)
	require.NoError(t, err, errMsgWorkflowShouldReachTerminal)
	assert.Equal(t, "finished", resp.Status,
		"retry-test should succeed after retrying transient failures")
}

// TestParallelRetry_CompletesWithConcurrentRetries triggers a workflow with two
// parallel branches where one branch retries once before succeeding.
func (s *WorkflowResilienceSuite) TestParallelRetry_CompletesWithConcurrentRetries() {
	t := s.T()
	wfID := TriggerExampleWorkflow(t, s.client, s.baseURL, "parallel-retry-test")
	require.NotEmpty(t, wfID)

	resp, err := WaitForWorkflowTerminal(s.client, s.baseURL, wfID, FastStatusTimeout)
	require.NoError(t, err, errMsgWorkflowShouldReachTerminal)
	assert.Equal(t, "finished", resp.Status,
		"parallel-retry-test should finish after branch-b retries succeed")
}

// TestTimeout_FollowsRecoveryPath triggers a workflow where a node has a 1s execution
// timeout but its function takes 5s. The timeout fires and the error edge routes to recovery.
func (s *WorkflowResilienceSuite) TestTimeout_FollowsRecoveryPath() {
	t := s.T()
	wfID := TriggerExampleWorkflow(t, s.client, s.baseURL, "timeout-test")
	require.NotEmpty(t, wfID)

	resp, err := WaitForWorkflowTerminal(s.client, s.baseURL, wfID, DefaultStatusTimeout)
	require.NoError(t, err, errMsgWorkflowShouldReachTerminal)
	assert.Equal(t, "finished", resp.Status,
		"timeout-test should finish via the recovery error edge")
}
