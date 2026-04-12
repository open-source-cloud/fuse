package workflow_test

import (
	"testing"
	"time"

	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/stretchr/testify/assert"
)

func TestSchemaVersion_Fields(t *testing.T) {
	now := time.Now().UTC()
	schema := workflow.GraphSchema{ID: "test-schema", Name: "Test Schema"}

	sv := workflow.SchemaVersion{
		SchemaID:  "test-schema",
		Version:   3,
		Schema:    schema,
		CreatedAt: now,
		CreatedBy: "user@example.com",
		Comment:   "bug fix",
		IsActive:  true,
	}

	assert.Equal(t, "test-schema", sv.SchemaID)
	assert.Equal(t, 3, sv.Version)
	assert.Equal(t, "Test Schema", sv.Schema.Name)
	assert.Equal(t, now, sv.CreatedAt)
	assert.Equal(t, "user@example.com", sv.CreatedBy)
	assert.Equal(t, "bug fix", sv.Comment)
	assert.True(t, sv.IsActive)
}

func TestSchemaVersionHistory_Fields(t *testing.T) {
	h := workflow.SchemaVersionHistory{
		SchemaID:      "my-schema",
		ActiveVersion: 2,
		LatestVersion: 3,
		TotalVersions: 3,
	}

	assert.Equal(t, "my-schema", h.SchemaID)
	assert.Equal(t, 2, h.ActiveVersion)
	assert.Equal(t, 3, h.LatestVersion)
	assert.Equal(t, 3, h.TotalVersions)
}
