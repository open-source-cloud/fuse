package actors

import (
	"testing"
	"time"

	"github.com/open-source-cloud/fuse/internal/events"
	"github.com/stretchr/testify/assert"
)

func TestBuildEventIdempotencyKey_Deterministic(t *testing.T) {
	event := events.Event{
		Type:      "workflow.completed",
		Source:    "wf-123",
		Timestamp: time.Now(),
		Data:      map[string]any{"schemaId": "my-schema", "status": "finished"},
	}

	key1 := buildEventIdempotencyKey("target-schema", event)
	key2 := buildEventIdempotencyKey("target-schema", event)

	assert.Equal(t, key1, key2, "same event should produce same key")
}

func TestBuildEventIdempotencyKey_DifferentSchemas(t *testing.T) {
	event := events.Event{
		Type:   "workflow.completed",
		Source: "wf-123",
		Data:   map[string]any{"status": "finished"},
	}

	key1 := buildEventIdempotencyKey("schema-a", event)
	key2 := buildEventIdempotencyKey("schema-b", event)

	assert.NotEqual(t, key1, key2)
}

func TestBuildEventIdempotencyKey_DifferentSources(t *testing.T) {
	event1 := events.Event{
		Type:   "workflow.completed",
		Source: "wf-111",
		Data:   map[string]any{"status": "finished"},
	}
	event2 := events.Event{
		Type:   "workflow.completed",
		Source: "wf-222",
		Data:   map[string]any{"status": "finished"},
	}

	key1 := buildEventIdempotencyKey("schema-a", event1)
	key2 := buildEventIdempotencyKey("schema-a", event2)

	assert.NotEqual(t, key1, key2)
}

func TestBuildEventIdempotencyKey_DifferentData(t *testing.T) {
	event1 := events.Event{
		Type:   "workflow.completed",
		Source: "wf-123",
		Data:   map[string]any{"version": 1},
	}
	event2 := events.Event{
		Type:   "workflow.completed",
		Source: "wf-123",
		Data:   map[string]any{"version": 2},
	}

	key1 := buildEventIdempotencyKey("schema-a", event1)
	key2 := buildEventIdempotencyKey("schema-a", event2)

	assert.NotEqual(t, key1, key2)
}
