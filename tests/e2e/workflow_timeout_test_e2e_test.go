//go:build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestE2E_ExampleWorkflow_timeout_test(t *testing.T) {
	client, base := RequireE2E(t)
	dir := WorkflowsDirForTests(t)
	const schemaID = "timeout-test"

	// Arrange — JSON schema lives at examples/workflows/timeout-test.json

	// Act
	wfID := UpsertAndTriggerExampleWorkflow(t, client, base, dir, schemaID)

	// Assert
	require.NotEmpty(t, wfID)
}
