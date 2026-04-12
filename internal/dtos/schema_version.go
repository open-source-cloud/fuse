package dtos

import (
	"time"

	"github.com/open-source-cloud/fuse/internal/workflow"
)

// SchemaVersionSummary is a lightweight version list item (no schema body).
type SchemaVersionSummary struct {
	Version   int       `json:"version" example:"2"`
	IsActive  bool      `json:"isActive" example:"true"`
	CreatedAt time.Time `json:"createdAt" example:"2025-07-01T00:00:00Z"`
	CreatedBy string    `json:"createdBy,omitempty" example:"user@example.com"`
	Comment   string    `json:"comment,omitempty" example:"Fix conditional edge"`
}

// SchemaVersionListResponse is the response for GET /v1/schemas/{schemaID}/versions.
type SchemaVersionListResponse struct {
	SchemaID      string                 `json:"schemaId" example:"my-workflow"`
	ActiveVersion int                    `json:"activeVersion" example:"3"`
	LatestVersion int                    `json:"latestVersion" example:"3"`
	Versions      []SchemaVersionSummary `json:"versions"`
}

// SchemaVersionResponse is the response for GET /v1/schemas/{schemaID}/versions/{version}.
type SchemaVersionResponse struct {
	SchemaID  string               `json:"schemaId" example:"my-workflow"`
	Version   int                  `json:"version" example:"2"`
	Schema    workflow.GraphSchema `json:"schema"`
	IsActive  bool                 `json:"isActive" example:"false"`
	CreatedAt time.Time            `json:"createdAt" example:"2025-06-15T00:00:00Z"`
	CreatedBy string               `json:"createdBy,omitempty"`
	Comment   string               `json:"comment,omitempty"`
}

// ActivateVersionResponse is the response for POST /v1/schemas/{schemaID}/versions/{version}/activate.
type ActivateVersionResponse struct {
	SchemaID        string `json:"schemaId" example:"my-workflow"`
	ActiveVersion   int    `json:"activeVersion" example:"2"`
	PreviousVersion int    `json:"previousVersion" example:"3"`
}

// RollbackRequest is the request body for POST /v1/schemas/{schemaID}/rollback.
type RollbackRequest struct {
	Version int    `json:"version" validate:"required,min=1"`
	Comment string `json:"comment,omitempty"`
}

// RollbackResponse is the response for POST /v1/schemas/{schemaID}/rollback.
type RollbackResponse struct {
	SchemaID     string `json:"schemaId" example:"my-workflow"`
	NewVersion   int    `json:"newVersion" example:"4"`
	RestoredFrom int    `json:"restoredFrom" example:"1"`
}

// ToSchemaVersionSummary converts a workflow.SchemaVersion to a SchemaVersionSummary DTO.
func ToSchemaVersionSummary(sv workflow.SchemaVersion) SchemaVersionSummary {
	return SchemaVersionSummary{
		Version:   sv.Version,
		IsActive:  sv.IsActive,
		CreatedAt: sv.CreatedAt,
		CreatedBy: sv.CreatedBy,
		Comment:   sv.Comment,
	}
}

// ToSchemaVersionResponse converts a workflow.SchemaVersion to a SchemaVersionResponse DTO.
func ToSchemaVersionResponse(sv workflow.SchemaVersion) SchemaVersionResponse {
	return SchemaVersionResponse{
		SchemaID:  sv.SchemaID,
		Version:   sv.Version,
		Schema:    sv.Schema,
		IsActive:  sv.IsActive,
		CreatedAt: sv.CreatedAt,
		CreatedBy: sv.CreatedBy,
		Comment:   sv.Comment,
	}
}
