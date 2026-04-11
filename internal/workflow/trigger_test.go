package workflow

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTriggerConfig_JSON_HTTP(t *testing.T) {
	cfg := TriggerConfig{Type: TriggerHTTP}
	data, err := json.Marshal(cfg)
	require.NoError(t, err)

	var parsed TriggerConfig
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, TriggerHTTP, parsed.Type)
	assert.Nil(t, parsed.Cron)
	assert.Nil(t, parsed.Webhook)
	assert.Nil(t, parsed.Event)
}

func TestTriggerConfig_JSON_Cron(t *testing.T) {
	cfg := TriggerConfig{
		Type: TriggerCron,
		Cron: &CronConfig{
			Expression: "0 */5 * * *",
			Timezone:   "America/New_York",
			Input:      map[string]any{"key": "value"},
		},
	}

	data, err := json.Marshal(cfg)
	require.NoError(t, err)

	var parsed TriggerConfig
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, TriggerCron, parsed.Type)
	require.NotNil(t, parsed.Cron)
	assert.Equal(t, "0 */5 * * *", parsed.Cron.Expression)
	assert.Equal(t, "America/New_York", parsed.Cron.Timezone)
	assert.Equal(t, "value", parsed.Cron.Input["key"])
}

func TestTriggerConfig_JSON_Webhook(t *testing.T) {
	cfg := TriggerConfig{
		Type: TriggerWebhook,
		Webhook: &WebhookConfig{
			Path:   "/hooks/github-push",
			Method: "POST",
			Secret: "my-secret",
		},
	}

	data, err := json.Marshal(cfg)
	require.NoError(t, err)

	var parsed TriggerConfig
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, TriggerWebhook, parsed.Type)
	require.NotNil(t, parsed.Webhook)
	assert.Equal(t, "/hooks/github-push", parsed.Webhook.Path)
	assert.Equal(t, "my-secret", parsed.Webhook.Secret)
}

func TestTriggerConfig_JSON_Event(t *testing.T) {
	cfg := TriggerConfig{
		Type: TriggerEvent,
		Event: &EventConfig{
			EventType: "workflow.completed",
			Filter:    "data.schemaId == 'my-schema'",
		},
	}

	data, err := json.Marshal(cfg)
	require.NoError(t, err)

	var parsed TriggerConfig
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, TriggerEvent, parsed.Type)
	require.NotNil(t, parsed.Event)
	assert.Equal(t, "workflow.completed", parsed.Event.EventType)
	assert.Equal(t, "data.schemaId == 'my-schema'", parsed.Event.Filter)
}
