package workflow

import "github.com/go-playground/validator/v10"

type (
	// Package workflow function Package
	Package struct {
		ID        string              `json:"id" validate:"required"`
		Functions []*PackagedFunction `json:"functions" validate:"required,dive"`
		Tags      map[string]string   `json:"tags,omitempty"`
	}

	// PackagedFunction packaged Function
	PackagedFunction struct {
		ID       string            `json:"id" validate:"required"`
		Metadata FunctionMetadata  `json:"metadata" validate:"required"`
		Tags     map[string]string `json:"tags,omitempty"`
		Function Function          `json:"-"`
	}
)

// NewPackage creates a new Package
func NewPackage(id string, functions ...*PackagedFunction) *Package {
	return &Package{
		ID:        id,
		Functions: functions,
	}
}

// Validate validates the package and its functions
func (p *Package) Validate() error {
	validate := validator.New(validator.WithRequiredStructEnabled())
	return validate.Struct(p)
}

// NewFunction creates a new packaged Function
func NewFunction(id string, metadata FunctionMetadata, fn Function) *PackagedFunction {
	return &PackagedFunction{
		ID:       id,
		Metadata: metadata,
		Function: fn,
	}
}
