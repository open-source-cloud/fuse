package dtos

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Message string   `json:"message" example:"Internal server error"`
	Code    string   `json:"code" example:"INTERNAL_SERVER_ERROR"`
	Fields  []string `json:"fields" example:"field1,field2"`
}

// BadRequestError represents a 400 Bad Request error
type BadRequestError ErrorResponse

// NotFoundError represents a 404 Not Found error
type NotFoundError ErrorResponse

// InternalServerErrorResponse represents a 500 Internal Server Error
type InternalServerErrorResponse ErrorResponse

// ValidationErrorResponse represents a 400 Validation Error with field details
type ValidationErrorResponse ErrorResponse
