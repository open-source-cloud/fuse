//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// WorkflowPersistenceSuite verifies that workflows using PostgreSQL persistence
// correctly store state, journal entries, and can be queried after completion.
type WorkflowPersistenceSuite struct {
	suite.Suite
	client       *http.Client
	baseURL      string
	workflowsDir string
}

func TestWorkflowPersistenceSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(WorkflowPersistenceSuite))
}

func (s *WorkflowPersistenceSuite) SetupSuite() {
	s.client, s.baseURL = RequireE2E(s.T())
	s.workflowsDir = WorkflowsDirForTests(s.T())
	for _, id := range []string{"smallest-test", "small-test"} {
		UpsertSchema(s.T(), s.client, s.baseURL, s.workflowsDir, id)
	}
}

// TestTriggerAndFinish_PersistsState triggers a workflow via PG-backed FUSE
// and verifies the workflow reaches terminal state and can be queried.
func (s *WorkflowPersistenceSuite) TestTriggerAndFinish_PersistsState() {
	t := s.T()

	// Trigger a simple workflow
	wfID := TriggerExampleWorkflow(t, s.client, s.baseURL, "smallest-test")
	require.NotEmpty(t, wfID)

	// Wait for completion
	resp, err := WaitForWorkflowTerminal(s.client, s.baseURL, wfID, FastStatusTimeout)
	require.NoError(t, err, "workflow should reach terminal state")
	assert.Equal(t, "finished", resp.Status)

	// Query the workflow again — state must be persisted (not lost)
	status, err := GetWorkflowStatus(s.client, s.baseURL, wfID)
	require.NoError(t, err, "should be able to query completed workflow")
	assert.Equal(t, "finished", status.Status)
}

// TestMultipleWorkflows_IndependentState verifies that multiple workflows
// running concurrently maintain independent persisted state.
func (s *WorkflowPersistenceSuite) TestMultipleWorkflows_IndependentState() {
	t := s.T()

	// Trigger two workflows from different schemas
	wfID1 := TriggerExampleWorkflow(t, s.client, s.baseURL, "smallest-test")
	wfID2 := TriggerExampleWorkflow(t, s.client, s.baseURL, "small-test")
	require.NotEmpty(t, wfID1)
	require.NotEmpty(t, wfID2)
	require.NotEqual(t, wfID1, wfID2)

	// Both should finish independently
	resp1, err := WaitForWorkflowTerminal(s.client, s.baseURL, wfID1, FastStatusTimeout)
	require.NoError(t, err)
	assert.Equal(t, "finished", resp1.Status)

	resp2, err := WaitForWorkflowTerminal(s.client, s.baseURL, wfID2, FastStatusTimeout)
	require.NoError(t, err)
	assert.Equal(t, "finished", resp2.Status)
}

// TestSchemaRoundTrip_PersistsInDB verifies that PUT + GET schema goes through PG.
func (s *WorkflowPersistenceSuite) TestSchemaRoundTrip_PersistsInDB() {
	t := s.T()

	// PUT schema
	schemaID := "smallest-test"
	putURL := fmt.Sprintf("%s/v1/schemas/%s", s.baseURL, schemaID)
	file := s.workflowsDir + "/" + schemaID + ".json"
	payload, err := ReadSchemaFile(s.workflowsDir, file)
	require.NoError(t, err)

	code, err := PUTJSON(s.client, putURL, payload)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, code)

	// GET schema — must return from PG
	getCode, getBody, err := GET(s.client, putURL)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, getCode)

	var schema struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(getBody, &schema))
	assert.Equal(t, schemaID, schema.ID)
}

// TestPackageRegistration_PersistsInDB verifies that packages are persisted through PG.
func (s *WorkflowPersistenceSuite) TestPackageRegistration_PersistsInDB() {
	t := s.T()

	// GET packages — internal packages should be registered
	code, body, err := GET(s.client, s.baseURL+"/v1/packages")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, code)

	var resp struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
		Metadata struct {
			Total int `json:"total"`
		} `json:"metadata"`
	}
	require.NoError(t, json.Unmarshal(body, &resp))
	assert.Greater(t, resp.Metadata.Total, 0, "internal packages should be registered in PG")
}
