package workflow

import (
	"encoding/json"
	"sync"

	"github.com/go-playground/validator/v10"
)

type (
	// Package workflow function Package
	Package struct {
		mu        sync.RWMutex        `json:"-"`
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

// Encode encodes a package to a JSON byte array
func (p *Package) Encode() ([]byte, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(p)
}

// DecodePackage decodes a package from a JSON byte array
func DecodePackage(data []byte) (*Package, error) {
	var pkg Package
	err := json.Unmarshal(data, &pkg)
	if err != nil {
		return nil, err
	}
	// Validate the package after decoding
	if err := pkg.Validate(); err != nil {
		return nil, err
	}
	return &pkg, nil
}

// Validate validates the package and its functions
func (p *Package) Validate() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
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
