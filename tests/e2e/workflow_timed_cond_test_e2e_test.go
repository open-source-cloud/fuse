//go:build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestE2E_ExampleWorkflow_timed_cond_test(t *testing.T) {
	client, base := RequireE2E(t)
	dir := WorkflowsDirForTests(t)
	const schemaID = "timed-cond-test"

	// Arrange — JSON schema lives at examples/workflows/timed-cond-test.json

	// Act
	wfID := UpsertAndTriggerExampleWorkflow(t, client, base, dir, schemaID)

	// Assert
	require.NotEmpty(t, wfID)
}
