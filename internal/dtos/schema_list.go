package dtos

// GraphSchemaSummaryDTO is a compact schema row for list APIs.
type GraphSchemaSummaryDTO struct {
	SchemaID string `json:"schemaID" example:"my-workflow"`
	Name     string `json:"name" example:"My workflow"`
}

// SchemaListResponse is the response body for GET /v1/schemas.
type SchemaListResponse struct {
	Metadata PaginationMetadata      `json:"metadata"`
	Items    []GraphSchemaSummaryDTO `json:"items"`
}
