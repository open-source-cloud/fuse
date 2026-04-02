//go:build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestE2E_ExampleWorkflow_merge_strategy_test(t *testing.T) {
	client, base := RequireE2E(t)
	dir := WorkflowsDirForTests(t)
	const schemaID = "merge-strategy-test"

	// Arrange — JSON schema lives at examples/workflows/merge-strategy-test.json

	// Act
	wfID := UpsertAndTriggerExampleWorkflow(t, client, base, dir, schemaID)

	// Assert
	require.NotEmpty(t, wfID)
}
