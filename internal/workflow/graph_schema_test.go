package workflow

import (
	"encoding/json"
	"testing"

	pkgworkflow "github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGraphSchema_Clone_CopiesConcurrency(t *testing.T) {
	original := GraphSchema{
		ID:   "test",
		Name: "Test",
		Nodes: []*NodeSchema{
			{ID: "n1", Function: "debug/print"},
		},
		Edges: []*EdgeSchema{},
		Concurrency: &pkgworkflow.ConcurrencyConfig{
			Limit: 5,
			Key:   "input.userId",
		},
	}

	clone := original.Clone()

	// Verify deep copy
	assert.Equal(t, original.Concurrency.Limit, clone.Concurrency.Limit)
	assert.Equal(t, original.Concurrency.Key, clone.Concurrency.Key)

	// Modify original — should not affect clone
	original.Concurrency.Limit = 99
	assert.Equal(t, 5, clone.Concurrency.Limit)
}

func TestGraphSchema_Clone_CopiesTriggerConfig(t *testing.T) {
	original := GraphSchema{
		ID:   "test",
		Name: "Test",
		Nodes: []*NodeSchema{
			{ID: "n1", Function: "debug/print"},
		},
		Edges: []*EdgeSchema{},
		TriggerConfig: &TriggerConfig{
			Type: TriggerCron,
			Cron: &CronConfig{Expression: "0 */5 * * *"},
		},
	}

	clone := original.Clone()

	assert.Equal(t, TriggerCron, clone.TriggerConfig.Type)
	require.NotNil(t, clone.TriggerConfig.Cron)
	assert.Equal(t, "0 */5 * * *", clone.TriggerConfig.Cron.Expression)
}

func TestGraphSchema_Clone_NilOptionalFields(t *testing.T) {
	original := GraphSchema{
		ID:   "test",
		Name: "Test",
		Nodes: []*NodeSchema{
			{ID: "n1", Function: "debug/print"},
		},
		Edges: []*EdgeSchema{},
	}

	clone := original.Clone()

	assert.Nil(t, clone.Concurrency)
	assert.Nil(t, clone.TriggerConfig)
	assert.Nil(t, clone.Timeout)
}

func TestGraphSchema_JSON_WithTriggerConfig(t *testing.T) {
	schema := GraphSchema{
		ID:   "test",
		Name: "Test Schema",
		Nodes: []*NodeSchema{
			{ID: "n1", Function: "debug/print"},
		},
		Edges: []*EdgeSchema{},
		TriggerConfig: &TriggerConfig{
			Type: TriggerWebhook,
			Webhook: &WebhookConfig{
				Path:   "/hooks/test",
				Method: "POST",
			},
		},
		Concurrency: &pkgworkflow.ConcurrencyConfig{
			Limit: 10,
		},
	}

	data, err := json.Marshal(schema)
	require.NoError(t, err)

	var parsed GraphSchema
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	require.NotNil(t, parsed.TriggerConfig)
	assert.Equal(t, TriggerWebhook, parsed.TriggerConfig.Type)
	assert.Equal(t, "/hooks/test", parsed.TriggerConfig.Webhook.Path)
	require.NotNil(t, parsed.Concurrency)
	assert.Equal(t, 10, parsed.Concurrency.Limit)
}

func TestGraphSchema_JSON_BackwardCompatible(t *testing.T) {
	// Existing schema JSON without new fields should deserialize fine
	jsonData := `{"id":"old-schema","name":"Old","nodes":[{"id":"n1","function":"debug/print"}],"edges":[]}`

	var schema GraphSchema
	err := json.Unmarshal([]byte(jsonData), &schema)
	require.NoError(t, err)

	assert.Equal(t, "old-schema", schema.ID)
	assert.Nil(t, schema.TriggerConfig)
	assert.Nil(t, schema.Concurrency)
}
