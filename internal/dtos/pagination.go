package dtos

type (
	// PaginationMetadata is the metadata for a paginated response
	PaginationMetadata struct {
		Total int `json:"total"`
		Page  int `json:"page"`
		Size  int `json:"size"`
	}
	// PaginatedResponse is the response for a paginated request
	PaginatedResponse[T any] struct {
		Items      []T                `json:"items"`
		Pagination PaginationMetadata `json:"pagination"`
	}
)

// NewPaginatedResponse creates a new paginated response
func NewPaginatedResponse[T any](items []T, metadata PaginationMetadata) *PaginatedResponse[T] {
	return &PaginatedResponse[T]{
		Items:      items,
		Pagination: metadata,
	}
}
