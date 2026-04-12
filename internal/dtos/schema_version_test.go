package dtos_test

import (
	"testing"
	"time"

	"github.com/open-source-cloud/fuse/internal/dtos"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/stretchr/testify/assert"
)

func TestToSchemaVersionSummary(t *testing.T) {
	now := time.Now().UTC()
	sv := workflow.SchemaVersion{
		SchemaID:  "my-schema",
		Version:   3,
		Schema:    workflow.GraphSchema{ID: "my-schema"},
		CreatedAt: now,
		CreatedBy: "ops@example.com",
		Comment:   "hotfix",
		IsActive:  true,
	}

	dto := dtos.ToSchemaVersionSummary(sv)

	assert.Equal(t, 3, dto.Version)
	assert.True(t, dto.IsActive)
	assert.Equal(t, now, dto.CreatedAt)
	assert.Equal(t, "ops@example.com", dto.CreatedBy)
	assert.Equal(t, "hotfix", dto.Comment)
}

func TestToSchemaVersionResponse(t *testing.T) {
	now := time.Now().UTC()
	schema := workflow.GraphSchema{ID: "my-schema", Name: "My Schema"}
	sv := workflow.SchemaVersion{
		SchemaID:  "my-schema",
		Version:   2,
		Schema:    schema,
		CreatedAt: now,
		CreatedBy: "dev@example.com",
		Comment:   "feature",
		IsActive:  false,
	}

	dto := dtos.ToSchemaVersionResponse(sv)

	assert.Equal(t, "my-schema", dto.SchemaID)
	assert.Equal(t, 2, dto.Version)
	assert.Equal(t, "My Schema", dto.Schema.Name)
	assert.False(t, dto.IsActive)
	assert.Equal(t, now, dto.CreatedAt)
	assert.Equal(t, "dev@example.com", dto.CreatedBy)
	assert.Equal(t, "feature", dto.Comment)
}

func TestToSchemaVersionSummary_EmptyOptionals(t *testing.T) {
	sv := workflow.SchemaVersion{
		SchemaID: "s", Version: 1, IsActive: false, CreatedAt: time.Now().UTC(),
	}
	dto := dtos.ToSchemaVersionSummary(sv)
	assert.Empty(t, dto.CreatedBy)
	assert.Empty(t, dto.Comment)
}
