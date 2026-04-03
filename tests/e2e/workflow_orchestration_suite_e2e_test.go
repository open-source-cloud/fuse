//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// WorkflowOrchestrationSuite verifies advanced orchestration features:
// sleep/delay, sub-workflows, awakeables (external events), merge strategies,
// and timed conditions.
type WorkflowOrchestrationSuite struct {
	suite.Suite
	client       *http.Client
	baseURL      string
	workflowsDir string
}

func TestWorkflowOrchestrationSuite(t *testing.T) {
	suite.Run(t, new(WorkflowOrchestrationSuite))
}

func (s *WorkflowOrchestrationSuite) SetupSuite() {
	s.client, s.baseURL = RequireE2E(s.T())
	s.workflowsDir = WorkflowsDirForTests(s.T())
}

// TestSleep_CompletesAfterDelay triggers a workflow that sleeps for 5 seconds
// before completing. Uses a longer timeout to accommodate the sleep duration.
func (s *WorkflowOrchestrationSuite) TestSleep_CompletesAfterDelay() {
	t := s.T()
	wfID := UpsertAndTriggerExampleWorkflow(t, s.client, s.baseURL, s.workflowsDir, "sleep-test")
	require.NotEmpty(t, wfID)

	resp, err := WaitForWorkflowTerminal(s.client, s.baseURL, wfID, LongStatusTimeout)
	require.NoError(t, err, "workflow should complete after sleep")
	assert.Equal(t, "finished", resp.Status,
		"sleep-test should finish after the 5s sleep elapses")
}

// TestSubWorkflow_CompletesWithChild triggers a parent workflow that spawns
// a synchronous child (smallest-test) and waits for it to complete.
func (s *WorkflowOrchestrationSuite) TestSubWorkflow_CompletesWithChild() {
	t := s.T()
	wfID := UpsertAndTriggerExampleWorkflow(t, s.client, s.baseURL, s.workflowsDir, "subworkflow-test")
	require.NotEmpty(t, wfID)

	resp, err := WaitForWorkflowTerminal(s.client, s.baseURL, wfID, LongStatusTimeout)
	require.NoError(t, err, "parent workflow should reach terminal state")
	assert.Equal(t, "finished", resp.Status,
		"subworkflow-test should finish after child workflow completes")
}

// TestAwakeable_EntersSleepingState triggers a workflow with a system/wait node
// that blocks until an external event resolves it. Without resolving, the workflow
// should enter and remain in "sleeping" state.
func (s *WorkflowOrchestrationSuite) TestAwakeable_EntersSleepingState() {
	t := s.T()
	wfID := UpsertAndTriggerExampleWorkflow(t, s.client, s.baseURL, s.workflowsDir, "awakeable-test")
	require.NotEmpty(t, wfID)

	resp, err := WaitForWorkflowStatus(s.client, s.baseURL, wfID, "sleeping", DefaultStatusTimeout)
	require.NoError(t, err, "workflow should enter sleeping state while waiting for external event")
	assert.Equal(t, "sleeping", resp.Status,
		"awakeable-test should be sleeping until the awakeable is resolved")
}

// TestMergeStrategy_JoinsParallelBranches triggers a workflow with two parallel
// branches that merge into a single join node using a keyed merge strategy.
func (s *WorkflowOrchestrationSuite) TestMergeStrategy_JoinsParallelBranches() {
	t := s.T()
	wfID := UpsertAndTriggerExampleWorkflow(t, s.client, s.baseURL, s.workflowsDir, "merge-strategy-test")
	require.NotEmpty(t, wfID)

	resp, err := WaitForWorkflowTerminal(s.client, s.baseURL, wfID, DefaultStatusTimeout)
	require.NoError(t, err, "workflow should reach terminal state")
	assert.Equal(t, "finished", resp.Status,
		"merge-strategy-test should finish after both branches merge")
}

// TestTimedCondition_CompletesAfterTimer triggers a workflow with a 3s timer delay
// before conditional branching.
func (s *WorkflowOrchestrationSuite) TestTimedCondition_CompletesAfterTimer() {
	t := s.T()
	wfID := UpsertAndTriggerExampleWorkflow(t, s.client, s.baseURL, s.workflowsDir, "timed-cond-test")
	require.NotEmpty(t, wfID)

	resp, err := WaitForWorkflowTerminal(s.client, s.baseURL, wfID, LongStatusTimeout)
	require.NoError(t, err, "workflow should complete after timer delay")
	assert.Equal(t, "finished", resp.Status,
		"timed-cond-test should finish after the 3s timer and conditional evaluation")
}
