//go:build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestE2E_ExampleWorkflow_parallel_retry_test(t *testing.T) {
	client, base := RequireE2E(t)
	dir := WorkflowsDirForTests(t)
	const schemaID = "parallel-retry-test"

	// Arrange — JSON schema lives at examples/workflows/parallel-retry-test.json

	// Act
	wfID := UpsertAndTriggerExampleWorkflow(t, client, base, dir, schemaID)

	// Assert
	require.NotEmpty(t, wfID)
}
