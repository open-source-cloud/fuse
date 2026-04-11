//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// UpsertSchema PUTs the JSON schema without triggering it.
// If an e2e overlay variant exists, it is used instead of the default schema.
func UpsertSchema(t *testing.T, client *http.Client, baseURL, workflowsDir, schemaID string) {
	t.Helper()
	overlay := E2EOverlayDir(t)
	var body []byte
	var err error
	if overlay != "" {
		body, err = ReadSchemaFileWithOverlay(overlay, workflowsDir, schemaID)
	} else {
		file := filepath.Join(workflowsDir, schemaID+".json")
		body, err = ReadSchemaFile(workflowsDir, file)
	}
	require.NoError(t, err, "read schema %s", schemaID)

	putURL := fmt.Sprintf("%s/v1/schemas/%s", baseURL, schemaID)
	code, err := PUTJSON(client, putURL, body)
	require.NoError(t, err, "PUT schema %s", schemaID)
	require.GreaterOrEqual(t, code, 200, "PUT %s lower bound", schemaID)
	require.Less(t, code, 300, "PUT %s upper bound", schemaID)
}

// TriggerExampleWorkflow POSTs the trigger for an already-upserted schema; returns workflowId.
func TriggerExampleWorkflow(t *testing.T, client *http.Client, baseURL, schemaID string) string {
	t.Helper()
	triggerURL := fmt.Sprintf("%s/v1/workflows/trigger", baseURL)
	reqBody, err := MarshalTriggerBody(schemaID)
	require.NoError(t, err, "marshal trigger for %s", schemaID)
	postCode, respBody, err := POSTJSON(client, triggerURL, reqBody)
	require.NoError(t, err, "POST trigger %s", schemaID)
	require.GreaterOrEqual(t, postCode, 200, "trigger %s lower bound", schemaID)
	require.Less(t, postCode, 300, "trigger %s upper bound", schemaID)

	var tr TriggerResponse
	require.NoError(t, json.Unmarshal(respBody, &tr), "decode trigger body: %s", string(respBody))
	require.NotEmpty(t, tr.WorkflowID, "workflowId for %s", schemaID)
	return tr.WorkflowID
}

// UpsertAndTriggerExampleWorkflow PUTs the JSON schema and POSTs trigger; returns workflowId.
func UpsertAndTriggerExampleWorkflow(t *testing.T, client *http.Client, baseURL, workflowsDir, schemaID string) string {
	t.Helper()
	UpsertSchema(t, client, baseURL, workflowsDir, schemaID)
	return TriggerExampleWorkflow(t, client, baseURL, schemaID)
}
