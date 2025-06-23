package workflow

import (
	"errors"
	"github.com/go-playground/validator/v10"
)

var (
	// ErrGraphIDIsEmpty return when ID is empty
	ErrGraphIDIsEmpty error = errors.New("ID is empty")
)

// GraphSchema represents a data structure containing nodes and edges, identified by a unique ID and optionally named.
type GraphSchema struct {
	ID       string            `json:"id" validate:"required,uuid"`
	Name     string            `json:"name" validate:"required,lte=100"`
	Nodes    []*NodeSchema     `json:"nodes" validate:"required,dive"`
	Edges    []*EdgeSchema     `json:"edges" validate:"required,dive"`
	Version  int               `json:"version" validate:"required,gte=1"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Tags     map[string]string `json:"tags,omitempty"`
}

// IsVersion checks if the target version is equal to the actual version
func (gs *GraphSchema) IsVersion(version int) bool {
	return gs.Version == version
}

// IncrVersion increases the version
func (gs *GraphSchema) IncrVersion() {
	gs.Version++
}

// Validate validates the graph schema
func (f *GraphSchema) Validate() error {
	validate := validator.New(validator.WithRequiredStructEnabled())
	return validate.Struct(f)
}

// SetID sets the id of the schema
func (f *GraphSchema) SetID(id string) {
	f.ID = id
}
