//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// untriggeredReTriggerAttempts bounds how many times TriggerAndWaitTerminal re-triggers a
// workflow that gets stuck in "untriggered".
const untriggeredReTriggerAttempts = 3

// TriggerAndWaitTerminal triggers schemaID and waits for a terminal state, returning the final
// status.
//
// It works around an intermittent e2e issue where a freshly-triggered workflow can remain in
// "untriggered" — the trigger is accepted (a workflowId is returned) but the workflow actor is
// not spawned/claimed in time on the multi-node HA stack. This is a pre-existing flake unrelated
// to workflow logic (observed on main across unrelated workflows, and on the same commit both
// passing and failing); tracked separately for a proper root-cause. Only the stuck-"untriggered"
// state is retried by re-triggering (each trigger yields a fresh workflowId); a workflow that
// reaches any terminal state — including "error" — is returned as-is so real failures still surface.
func TriggerAndWaitTerminal(t *testing.T, client *http.Client, baseURL, schemaID string, timeout time.Duration) (string, *WorkflowStatusResponse) {
	t.Helper()
	var wfID string
	var resp *WorkflowStatusResponse
	var err error
	for attempt := 1; attempt <= untriggeredReTriggerAttempts; attempt++ {
		wfID = TriggerExampleWorkflow(t, client, baseURL, schemaID)
		resp, err = WaitForWorkflowTerminal(client, baseURL, wfID, timeout)
		if err == nil {
			return wfID, resp
		}
		if resp == nil || resp.Status != "untriggered" {
			break
		}
		t.Logf("e2e: workflow %s for %q stuck in 'untriggered' (attempt %d/%d); re-triggering [tracked flake]",
			wfID, schemaID, attempt, untriggeredReTriggerAttempts)
	}
	require.NoError(t, err, "workflow %s should reach terminal state", schemaID)
	return wfID, resp
}

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
