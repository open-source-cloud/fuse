//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_GET_workflow_trace(t *testing.T) {
	t.Parallel()
	client, base := RequireE2E(t)
	dir := WorkflowsDirForTests(t)

	// Arrange — upload schema and trigger a workflow
	schemaID := "smallest-test"
	file := filepath.Join(dir, schemaID+".json")
	payload, err := ReadSchemaFile(dir, file)
	require.NoError(t, err)
	putCode, err := PUTJSON(client, fmt.Sprintf("%s/v1/schemas/%s", base, schemaID), payload)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, putCode)

	triggerBody, _ := MarshalTriggerBody(schemaID)
	code, respBody, err := POSTJSON(client, base+"/v1/workflows/trigger", triggerBody)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, code)
	var triggerResp TriggerResponse
	require.NoError(t, json.Unmarshal(respBody, &triggerResp))

	// Wait for workflow to complete
	_, err = WaitForWorkflowTerminal(client, base, triggerResp.WorkflowID, DefaultStatusTimeout)
	require.NoError(t, err)

	// Act — get the trace
	traceURL := fmt.Sprintf("%s/v1/workflows/%s/trace", base, triggerResp.WorkflowID)
	traceCode, traceBody, err := GET(client, traceURL)
	require.NoError(t, err)

	// Assert
	require.Equal(t, http.StatusOK, traceCode, "body=%s", string(traceBody))
	var trace struct {
		WorkflowID string `json:"workflowId"`
		SchemaID   string `json:"schemaId"`
		Status     string `json:"status"`
		Steps      []struct {
			ExecID string `json:"execId"`
			Status string `json:"status"`
		} `json:"steps"`
	}
	require.NoError(t, json.Unmarshal(traceBody, &trace))
	assert.Equal(t, triggerResp.WorkflowID, trace.WorkflowID)
	assert.Equal(t, schemaID, trace.SchemaID)
	assert.Equal(t, "finished", trace.Status)
	assert.NotEmpty(t, trace.Steps)
}

func TestE2E_GET_workflow_trace_notFound(t *testing.T) {
	t.Parallel()
	client, base := RequireE2E(t)

	url := base + "/v1/workflows/nonexistent-workflow-id/trace"
	code, _, err := GET(client, url)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, code)
}

func TestE2E_GET_schema_traces(t *testing.T) {
	t.Parallel()
	client, base := RequireE2E(t)
	dir := WorkflowsDirForTests(t)

	// Arrange — upload schema and trigger a workflow
	schemaID := "smallest-test"
	file := filepath.Join(dir, schemaID+".json")
	payload, err := ReadSchemaFile(dir, file)
	require.NoError(t, err)
	_, _ = PUTJSON(client, fmt.Sprintf("%s/v1/schemas/%s", base, schemaID), payload)

	triggerBody, _ := MarshalTriggerBody(schemaID)
	_, respBody, _ := POSTJSON(client, base+"/v1/workflows/trigger", triggerBody)
	var triggerResp TriggerResponse
	_ = json.Unmarshal(respBody, &triggerResp)

	_, _ = WaitForWorkflowTerminal(client, base, triggerResp.WorkflowID, DefaultStatusTimeout)

	// Act — list traces for schema
	tracesURL := fmt.Sprintf("%s/v1/schemas/%s/traces?limit=10", base, schemaID)
	code, tracesBody, err := GET(client, tracesURL)
	require.NoError(t, err)

	// Assert
	require.Equal(t, http.StatusOK, code, "body=%s", string(tracesBody))
	var resp struct {
		Traces []struct {
			WorkflowID string `json:"workflowId"`
			Status     string `json:"status"`
		} `json:"traces"`
		Total int `json:"total"`
		Limit int `json:"limit"`
	}
	require.NoError(t, json.Unmarshal(tracesBody, &resp))
	assert.GreaterOrEqual(t, resp.Total, 1)
	assert.Equal(t, 10, resp.Limit)
}

func TestE2E_GET_schema_traces_empty(t *testing.T) {
	t.Parallel()
	client, base := RequireE2E(t)

	url := base + "/v1/schemas/nonexistent-schema/traces"
	code, body, err := GET(client, url)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, code)

	var resp struct {
		Traces []any `json:"traces"`
		Total  int   `json:"total"`
	}
	require.NoError(t, json.Unmarshal(body, &resp))
	assert.Equal(t, 0, resp.Total)
	assert.Empty(t, resp.Traces)
}
