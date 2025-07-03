package workflow

import (
	"encoding/json"
	"errors"
	"maps"

	"github.com/go-playground/validator/v10"
)

var (
	// ErrGraphIDIsEmpty return when ID is empty
	ErrGraphIDIsEmpty = errors.New("ID is empty")
)

// GraphSchema represents a data structure containing nodes and edges, identified by a unique ID and optionally named.
type GraphSchema struct {
	ID       string            `json:"id" validate:"required"`
	Name     string            `json:"name" validate:"required,lte=100"`
	Nodes    []*NodeSchema     `json:"nodes" validate:"required,dive"`
	Edges    []*EdgeSchema     `json:"edges" validate:"required,dive"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Tags     map[string]string `json:"tags,omitempty"`
}

// NewGraphSchemaFromJSON creates a new graph schema from a JSON specification
func NewGraphSchemaFromJSON(jsonSpec []byte) (*GraphSchema, error) {
	var schema *GraphSchema
	err := json.Unmarshal(jsonSpec, &schema)
	if err != nil {
		return nil, err
	}
	return schema, nil
}

// Validate validates the graph schema
func (f *GraphSchema) Validate() error {
	validate := validator.New(validator.WithRequiredStructEnabled())
	return validate.Struct(f)
}

// Clone clones the graph schema and returns a new instance
func (f *GraphSchema) Clone() GraphSchema {
	nodes := make([]*NodeSchema, len(f.Nodes))
	edges := make([]*EdgeSchema, len(f.Edges))

	for i, node := range f.Nodes {
		nodes[i] = node.Clone()
	}

	for i, edge := range f.Edges {
		edges[i] = edge.Clone()
	}

	return GraphSchema{
		ID:       f.ID,
		Name:     f.Name,
		Nodes:    nodes,
		Edges:    edges,
		Metadata: maps.Clone(f.Metadata),
		Tags:     maps.Clone(f.Tags),
	}
}
