//go:build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestE2E_ExampleWorkflow_github_request_example(t *testing.T) {
	client, base := RequireE2E(t)
	dir := WorkflowsDirForTests(t)
	const schemaID = "github-request-example"

	// Arrange — JSON schema lives at examples/workflows/github-request-example.json

	// Act
	wfID := UpsertAndTriggerExampleWorkflow(t, client, base, dir, schemaID)

	// Assert
	require.NotEmpty(t, wfID)
}
