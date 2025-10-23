package dtos

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Message string `json:"message" example:"Operation successful"`
	Code    string `json:"code" example:"OK"`
}

// PaginatedResponse wraps paginated data
type PaginatedResponse[T any] struct {
	Metadata PaginationMetadata `json:"metadata"`
	Items    []T                `json:"items"`
}

// PaginationMetadata represents pagination metadata
type PaginationMetadata struct {
	Total int `json:"total"`
	Page  int `json:"page"`
	Size  int `json:"size"`
}

// HealthCheckResponse represents health check response
type HealthCheckResponse struct {
	Message string `json:"message" example:"OK"`
}
