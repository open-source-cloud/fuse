// Package graph provides a memory graph implementation
package graph

import "github.com/open-source-cloud/fuse/internal/graph/schema"

type (
	// Edge describes a graph's edge
	Edge interface {
		ID() string
		IsConditional() bool
		Condition() *schema.EdgeCondition
		Input() []schema.InputMapping
		From() Node
		To() Node
	}

	edge struct {
		id        string
		condition *schema.EdgeCondition
		input     []schema.InputMapping
		from      Node
		to        Node
	}
)

// NewEdge creates and returns a new edge with the specified from and to nodes.
func NewEdge(id string, from Node, to Node, condition *schema.EdgeCondition, input []schema.InputMapping) Edge {
	return &edge{
		id:        id,
		condition: condition,
		input:     input,
		from:      from,
		to:        to,
	}
}

// ID returns the edge ID
func (e *edge) ID() string {
	return e.id
}

// IsConditional returns true if this edge has a conditional
func (e *edge) IsConditional() bool {
	return e.condition != nil
}

// Condition returns the edge conditional
func (e *edge) Condition() *schema.EdgeCondition {
	return e.condition
}

// Input returns the edge input mappings
func (e *edge) Input() []schema.InputMapping {
	return e.input
}

// From returns the edge from node
func (e *edge) From() Node {
	return e.from
}

// To return the edge to node
func (e *edge) To() Node {
	return e.to
}
