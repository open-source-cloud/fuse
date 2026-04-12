package workflow

import "time"

// SchemaVersion represents an immutable snapshot of a workflow schema at a specific version.
// Each call to PUT /v1/schemas/{schemaID} creates a new SchemaVersion instead of overwriting.
type SchemaVersion struct {
	SchemaID  string      `json:"schemaId"`
	Version   int         `json:"version"`
	Schema    GraphSchema `json:"schema"`
	CreatedAt time.Time   `json:"createdAt"`
	CreatedBy string      `json:"createdBy,omitempty"`
	Comment   string      `json:"comment,omitempty"`
	IsActive  bool        `json:"isActive"`
}

// SchemaVersionHistory summarises the versioning state for a schema.
type SchemaVersionHistory struct {
	SchemaID      string `json:"schemaId"`
	ActiveVersion int    `json:"activeVersion"`
	LatestVersion int    `json:"latestVersion"`
	TotalVersions int    `json:"totalVersions"`
}
