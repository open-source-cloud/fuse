package packages

import "errors"

var (
	// ErrLoadedPackageNotFound is returned when the loaded package is not found
	ErrLoadedPackageNotFound = errors.New("loaded package not found")
	// ErrLoadedFunctionNotFound is returned when the loaded function is not found
	ErrLoadedFunctionNotFound = errors.New("loaded function not found")
)
