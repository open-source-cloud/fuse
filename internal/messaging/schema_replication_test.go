package messaging_test

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/stretchr/testify/require"
)

func TestPublishGraphSchemaUpsertPayload(t *testing.T) {
	payload := messaging.GraphSchemaReplicationPayload{SchemaID: "s1", SchemaJSON: []byte(`{}`)}
	msg := messaging.NewPublishGraphSchemaUpsertMessage(payload)
	got, ok := msg.PublishGraphSchemaUpsertPayload()
	require.True(t, ok)
	require.Equal(t, payload.SchemaID, got.SchemaID)
	require.Equal(t, payload.SchemaJSON, got.SchemaJSON)
}
