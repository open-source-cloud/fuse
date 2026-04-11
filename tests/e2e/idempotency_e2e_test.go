//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_TriggerWorkflow_Idempotency_Deduplication(t *testing.T) {
	client, base := RequireE2E(t)

	// Arrange — ensure a schema exists
	dir := WorkflowsDirForTests(t)
	schemaID := "smallest-test"
	payload, err := ReadSchemaFile(dir, dir+"/"+schemaID+".json")
	require.NoError(t, err)
	putURL := base + "/v1/schemas/" + schemaID
	putCode, err := PUTJSON(client, putURL, payload)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, putCode)

	// Act — trigger with idempotency key
	idempotencyKey := "e2e-idemp-" + t.Name()
	triggerBody, err := json.Marshal(map[string]string{
		"schemaID":       schemaID,
		"idempotencyKey": idempotencyKey,
	})
	require.NoError(t, err)

	triggerURL := base + "/v1/workflows/trigger"
	code1, body1, err := POSTJSON(client, triggerURL, triggerBody)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, code1)

	var resp1 struct {
		WorkflowID   string `json:"workflowId"`
		Deduplicated bool   `json:"deduplicated"`
	}
	require.NoError(t, json.Unmarshal(body1, &resp1))
	assert.NotEmpty(t, resp1.WorkflowID)
	assert.False(t, resp1.Deduplicated)

	// Act — trigger again with same idempotency key
	code2, body2, err := POSTJSON(client, triggerURL, triggerBody)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, code2)

	var resp2 struct {
		WorkflowID   string `json:"workflowId"`
		Deduplicated bool   `json:"deduplicated"`
	}
	require.NoError(t, json.Unmarshal(body2, &resp2))

	// Assert — same workflow ID returned, marked as deduplicated
	assert.Equal(t, resp1.WorkflowID, resp2.WorkflowID)
	assert.True(t, resp2.Deduplicated)
}

func TestE2E_TriggerWorkflow_Idempotency_DifferentKeys(t *testing.T) {
	client, base := RequireE2E(t)

	// Arrange
	dir := WorkflowsDirForTests(t)
	schemaID := "smallest-test"
	payload, err := ReadSchemaFile(dir, dir+"/"+schemaID+".json")
	require.NoError(t, err)
	putURL := base + "/v1/schemas/" + schemaID
	_, _ = PUTJSON(client, putURL, payload)

	triggerURL := base + "/v1/workflows/trigger"

	// Act — trigger with different keys
	body1, _ := json.Marshal(map[string]string{
		"schemaID":       schemaID,
		"idempotencyKey": "key-A-" + t.Name(),
	})
	body2, _ := json.Marshal(map[string]string{
		"schemaID":       schemaID,
		"idempotencyKey": "key-B-" + t.Name(),
	})

	_, resp1, _ := POSTJSON(client, triggerURL, body1)
	_, resp2, _ := POSTJSON(client, triggerURL, body2)

	var r1, r2 struct {
		WorkflowID string `json:"workflowId"`
	}
	_ = json.Unmarshal(resp1, &r1)
	_ = json.Unmarshal(resp2, &r2)

	// Assert — different workflow IDs
	assert.NotEqual(t, r1.WorkflowID, r2.WorkflowID)
}

func TestE2E_TriggerWorkflow_NoIdempotencyKey(t *testing.T) {
	client, base := RequireE2E(t)

	// Arrange
	dir := WorkflowsDirForTests(t)
	schemaID := "smallest-test"
	payload, err := ReadSchemaFile(dir, dir+"/"+schemaID+".json")
	require.NoError(t, err)
	putURL := base + "/v1/schemas/" + schemaID
	_, _ = PUTJSON(client, putURL, payload)

	triggerURL := base + "/v1/workflows/trigger"
	body, _ := json.Marshal(map[string]string{"schemaID": schemaID})

	// Act — trigger twice without idempotency key
	_, resp1, _ := POSTJSON(client, triggerURL, body)
	_, resp2, _ := POSTJSON(client, triggerURL, body)

	var r1, r2 struct {
		WorkflowID string `json:"workflowId"`
	}
	_ = json.Unmarshal(resp1, &r1)
	_ = json.Unmarshal(resp2, &r2)

	// Assert — different workflow IDs (no dedup)
	assert.NotEqual(t, r1.WorkflowID, r2.WorkflowID)
}
