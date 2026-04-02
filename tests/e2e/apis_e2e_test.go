//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_GET_health(t *testing.T) {
	client, base := RequireE2E(t)

	// Arrange
	url := base + "/health"

	// Act
	code, body, err := GET(client, url)

	// Assert
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, code)
	var resp struct {
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(body, &resp))
	assert.Equal(t, "OK", resp.Message)
}

func TestE2E_GET_v1_packages(t *testing.T) {
	client, base := RequireE2E(t)

	// Arrange
	url := base + "/v1/packages"

	// Act
	code, body, err := GET(client, url)

	// Assert
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
	assert.GreaterOrEqual(t, resp.Metadata.Total, 0)
	assert.Len(t, resp.Items, resp.Metadata.Total)
}

func TestE2E_GET_v1_packages_byID(t *testing.T) {
	client, base := RequireE2E(t)

	// Arrange — pick first package from list
	listCode, listBody, err := GET(client, base+"/v1/packages")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, listCode)
	var list struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(listBody, &list))
	require.NotEmpty(t, list.Items, "need at least one registered package")
	packageID := list.Items[0].ID
	url := fmt.Sprintf("%s/v1/packages/%s", base, packageID)

	// Act
	code, body, err := GET(client, url)

	// Assert
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, code)
	var pkg struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(body, &pkg))
	assert.Equal(t, packageID, pkg.ID)
}

func TestE2E_PUT_GET_v1_schemas_roundTrip(t *testing.T) {
	client, base := RequireE2E(t)
	dir := WorkflowsDirForTests(t)

	// Arrange
	schemaID := "smallest-test"
	file := filepath.Join(dir, schemaID+".json")
	payload, err := ReadSchemaFile(dir, file)
	require.NoError(t, err)
	putURL := fmt.Sprintf("%s/v1/schemas/%s", base, schemaID)

	// Act — upsert
	putCode, err := PUTJSON(client, putURL, payload)
	require.NoError(t, err)

	// Assert — PUT
	require.Equal(t, http.StatusOK, putCode)

	// Act — fetch
	getCode, getBody, err := GET(client, putURL)

	// Assert — GET
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, getCode)
	var schema struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(getBody, &schema))
	assert.Equal(t, schemaID, schema.ID)
}

func TestE2E_PUT_v1_packages_roundTrip(t *testing.T) {
	client, base := RequireE2E(t)

	// Arrange — load an existing package and PUT it back (noop update)
	listCode, listBody, err := GET(client, base+"/v1/packages")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, listCode)
	var list struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(listBody, &list))
	require.NotEmpty(t, list.Items)
	packageID := list.Items[0].ID
	getURL := fmt.Sprintf("%s/v1/packages/%s", base, packageID)
	gc, pkgBody, err := GET(client, getURL)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, gc)

	// Act
	putCode, err := PUTJSON(client, getURL, pkgBody)

	// Assert
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, putCode)
}

func TestE2E_POST_v1_workflows_cancel(t *testing.T) {
	client, base := RequireE2E(t)

	// Arrange
	wfID := uuid.New().String()
	url := fmt.Sprintf("%s/v1/workflows/%s/cancel", base, wfID)
	body := []byte(`{"reason":"e2e"}`)

	// Act
	code, respBody, err := POSTJSON(client, url, body)

	// Assert
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, code)
	var resp struct {
		WorkflowID string `json:"workflowId"`
		Status     string `json:"status"`
	}
	require.NoError(t, json.Unmarshal(respBody, &resp))
	assert.Equal(t, wfID, resp.WorkflowID)
	assert.Equal(t, "cancelled", resp.Status)
}

func TestE2E_POST_v1_workflows_execs_asyncResult(t *testing.T) {
	client, base := RequireE2E(t)

	// Arrange — valid body; workflow handler may not exist → 500 is acceptable for smoke
	wfID := uuid.New().String()
	execID := "e2e-exec-" + uuid.New().String()[:8]
	url := fmt.Sprintf("%s/v1/workflows/%s/execs/%s", base, wfID, execID)
	body := []byte(`{"result":{"status":"success","data":{"ok":true}}}`)

	// Act
	code, respBody, err := POSTJSON(client, url, body)

	// Assert — endpoint accepts JSON; delivery may fail if no running workflow actor
	require.NoError(t, err)
	assert.Contains(t, []int{http.StatusOK, http.StatusInternalServerError}, code,
		"async result endpoint should respond; body=%s", string(respBody))
	if code == http.StatusOK {
		var resp struct {
			Code string `json:"code"`
		}
		require.NoError(t, json.Unmarshal(respBody, &resp))
		assert.Equal(t, "OK", resp.Code)
	}
}

func TestE2E_POST_v1_awakeables_resolve_notFound(t *testing.T) {
	client, base := RequireE2E(t)

	// Arrange
	url := base + "/v1/awakeables/e2e-nonexistent-awakeable/resolve"
	body := []byte(`{"data":{"k":"v"}}`)

	// Act
	code, respBody, err := POSTJSON(client, url, body)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, code, "body=%s", string(respBody))
}
