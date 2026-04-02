//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// WorkflowExecutionSuite verifies that core workflow execution completes
// successfully for linear pipelines, parallel fan-out, and conditional branching.
type WorkflowExecutionSuite struct {
	suite.Suite
	client       *http.Client
	baseURL      string
	workflowsDir string
}

func TestWorkflowExecutionSuite(t *testing.T) {
	suite.Run(t, new(WorkflowExecutionSuite))
}

func (s *WorkflowExecutionSuite) SetupSuite() {
	s.client, s.baseURL = RequireE2E(s.T())
	s.workflowsDir = WorkflowsDirForTests(s.T())
}

// triggerAndWaitFinished triggers a workflow and waits for it to reach "finished" status.
func (s *WorkflowExecutionSuite) triggerAndWaitFinished(schemaID string) {
	t := s.T()
	t.Helper()

	wfID := UpsertAndTriggerExampleWorkflow(t, s.client, s.baseURL, s.workflowsDir, schemaID)
	require.NotEmpty(t, wfID, "trigger should return a workflow ID")

	resp, err := WaitForWorkflowTerminal(s.client, s.baseURL, wfID, DefaultStatusTimeout)
	require.NoError(t, err, "workflow %s should reach terminal state", schemaID)
	assert.Equal(t, wfID, resp.WorkflowID, "response should echo back the workflow ID")
	assert.Equal(t, "finished", resp.Status, "workflow %s should finish successfully", schemaID)
}

// TestSmallestPipeline_Finishes runs a minimal 3-node pipeline (nil -> rand -> sum).
func (s *WorkflowExecutionSuite) TestSmallestPipeline_Finishes() {
	s.triggerAndWaitFinished("smallest-test")
}

// TestParallelFanOut_Finishes runs a pipeline with two parallel rand nodes feeding a sum.
func (s *WorkflowExecutionSuite) TestParallelFanOut_Finishes() {
	s.triggerAndWaitFinished("small-test")
}

// TestConditionalBranch_Finishes runs a workflow with if/else branching based on a threshold.
func (s *WorkflowExecutionSuite) TestConditionalBranch_Finishes() {
	s.triggerAndWaitFinished("small-cond-test")
}

// TestExpressionCondition_Finishes runs a workflow with expression-based conditional routing.
func (s *WorkflowExecutionSuite) TestExpressionCondition_Finishes() {
	s.triggerAndWaitFinished("expression-condition-test")
}

// TestDurableExecution_Finishes runs a multi-step durable pipeline (rand -> rand -> sum).
func (s *WorkflowExecutionSuite) TestDurableExecution_Finishes() {
	s.triggerAndWaitFinished("durable-test")
}

// TestSumRandBranch_Finishes runs a workflow with a 3s timer, three parallel rands, sum, and conditional branching.
func (s *WorkflowExecutionSuite) TestSumRandBranch_Finishes() {
	t := s.T()
	wfID := UpsertAndTriggerExampleWorkflow(t, s.client, s.baseURL, s.workflowsDir, "sum-rand-branch")
	require.NotEmpty(t, wfID)

	resp, err := WaitForWorkflowTerminal(s.client, s.baseURL, wfID, LongStatusTimeout)
	require.NoError(t, err, "workflow should reach terminal state")
	assert.Equal(t, "finished", resp.Status, "sum-rand-branch should finish successfully")
}
