package messaging

// GraphSchemaReplicationPayload is published via ergo SendEvent and applied on peer nodes without republishing.
type GraphSchemaReplicationPayload struct {
	SchemaID   string
	SchemaJSON []byte
}

// NewPublishGraphSchemaUpsertMessage wraps a replication payload for the schema replication actor.
func NewPublishGraphSchemaUpsertMessage(payload GraphSchemaReplicationPayload) Message {
	return Message{Type: PublishGraphSchemaUpsert, Args: payload}
}

// PublishGraphSchemaUpsertPayload extracts the payload from a PublishGraphSchemaUpsert message.
func (m Message) PublishGraphSchemaUpsertPayload() (GraphSchemaReplicationPayload, bool) {
	if m.Type != PublishGraphSchemaUpsert {
		return GraphSchemaReplicationPayload{}, false
	}
	p, ok := m.Args.(GraphSchemaReplicationPayload)
	return p, ok
}
