package workflow

import "errors"

var (
	// ErrInvalidFunctionFormat is returned when the function format is invalid
	ErrInvalidFunctionFormat = errors.New("invalid function format: must contain '/' to separate package and function")
)
