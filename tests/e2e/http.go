package e2e

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// DefaultAPIURL is the base URL when E2E_API_URL is unset.
	DefaultAPIURL = "http://localhost:9090"
	// HealthAttempts is how many times GET /health is tried before giving up.
	HealthAttempts = 30
	// HealthInterval is the sleep between health poll attempts.
	HealthInterval = 2 * time.Second
	// HTTPClientTimeout bounds each HTTP request duration.
	HTTPClientTimeout = 30 * time.Second
	// StatusPollInterval is the sleep between workflow status poll attempts.
	StatusPollInterval = 500 * time.Millisecond
	// DefaultStatusTimeout is the default timeout for waiting for a workflow status.
	DefaultStatusTimeout = 30 * time.Second
	// LongStatusTimeout is a longer timeout for workflows that sleep or do external calls.
	LongStatusTimeout = 45 * time.Second
)

// TriggerResponse is the subset of POST /v1/workflows/trigger JSON we assert on.
type TriggerResponse struct {
	WorkflowID string `json:"workflowId"`
}

// WorkflowStatusResponse is the JSON returned by GET /v1/workflows/{workflowID}.
type WorkflowStatusResponse struct {
	WorkflowID string `json:"workflowId"`
	Status     string `json:"status"`
}

// NewHTTPClient returns a client with a bounded timeout suitable for E2E requests.
// Each client gets its own Transport clone: the default nil Transport is http.DefaultTransport,
// which is shared by every client in the process, so idle connections from one test can be
// handed to the next and fail with EOF if the server already closed its side.
func NewHTTPClient() *http.Client {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.DisableKeepAlives = true
	return &http.Client{
		Timeout:   HTTPClientTimeout,
		Transport: tr,
	}
}

// WaitForHealth polls GET /health until success or attempts exhausted.
func WaitForHealth(client *http.Client, apiURL string) error {
	url := apiURL + "/health"
	for attempt := 0; attempt < HealthAttempts; attempt++ {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
		}
		time.Sleep(HealthInterval)
	}
	return errors.New("API did not become ready within timeout")
}

// PUTJSON performs PUT with application/json body; drains response body; returns status code.
func PUTJSON(client *http.Client, url string, body []byte) (int, error) {
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, nil
}

// GET performs GET and returns status and response body.
func GET(client *http.Client, url string) (int, []byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, respBody, nil
}

// POSTJSON performs POST with application/json body and returns status and response bytes.
func POSTJSON(client *http.Client, url string, body []byte) (int, []byte, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, respBody, nil
}

// MarshalTriggerBody returns JSON for POST /v1/workflows/trigger.
func MarshalTriggerBody(schemaID string) ([]byte, error) {
	return json.Marshal(map[string]string{"schemaID": schemaID})
}

// GetWorkflowStatus fetches the current status of a workflow.
func GetWorkflowStatus(client *http.Client, baseURL, workflowID string) (*WorkflowStatusResponse, error) {
	url := fmt.Sprintf("%s/v1/workflows/%s", baseURL, workflowID)
	code, body, err := GET(client, url)
	if err != nil {
		return nil, fmt.Errorf("GET workflow status: %w", err)
	}
	if code == http.StatusNotFound {
		return nil, fmt.Errorf("workflow %s not found (404)", workflowID)
	}
	if code != http.StatusOK {
		return nil, fmt.Errorf("GET workflow status: unexpected status %d, body=%s", code, string(body))
	}
	var resp WorkflowStatusResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode workflow status: %w", err)
	}
	return &resp, nil
}

// isTerminalStatus returns true if the workflow status is a terminal state.
func isTerminalStatus(status string) bool {
	switch status {
	case "finished", "error", "cancelled":
		return true
	}
	return false
}

// WaitForWorkflowTerminal polls GET /v1/workflows/{workflowID} until a terminal state
// (finished, error, cancelled) is reached or the timeout expires.
func WaitForWorkflowTerminal(client *http.Client, baseURL, workflowID string, timeout time.Duration) (*WorkflowStatusResponse, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := GetWorkflowStatus(client, baseURL, workflowID)
		if err != nil {
			time.Sleep(StatusPollInterval)
			continue
		}
		if isTerminalStatus(resp.Status) {
			return resp, nil
		}
		time.Sleep(StatusPollInterval)
	}
	// One final attempt to return whatever status we have.
	resp, err := GetWorkflowStatus(client, baseURL, workflowID)
	if err != nil {
		return nil, fmt.Errorf("workflow %s did not reach terminal state within %s: %w", workflowID, timeout, err)
	}
	return resp, fmt.Errorf("workflow %s still in status %q after %s", workflowID, resp.Status, timeout)
}

// WaitForWorkflowStatus polls GET /v1/workflows/{workflowID} until the given target status
// is reached or the timeout expires. Use this for non-terminal states like "sleeping".
func WaitForWorkflowStatus(client *http.Client, baseURL, workflowID, targetStatus string, timeout time.Duration) (*WorkflowStatusResponse, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := GetWorkflowStatus(client, baseURL, workflowID)
		if err != nil {
			time.Sleep(StatusPollInterval)
			continue
		}
		if resp.Status == targetStatus {
			return resp, nil
		}
		if isTerminalStatus(resp.Status) {
			return resp, fmt.Errorf("workflow %s reached terminal status %q while waiting for %q", workflowID, resp.Status, targetStatus)
		}
		time.Sleep(StatusPollInterval)
	}
	resp, err := GetWorkflowStatus(client, baseURL, workflowID)
	if err != nil {
		return nil, fmt.Errorf("workflow %s did not reach status %q within %s: %w", workflowID, targetStatus, timeout, err)
	}
	return resp, fmt.Errorf("workflow %s still in status %q after %s (expected %q)", workflowID, resp.Status, timeout, targetStatus)
}
