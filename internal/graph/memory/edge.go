// Package memory provides a memory graph implementation
package memory

import "github.com/open-source-cloud/fuse/pkg/graph"

// Edge is a memory graph edge
type Edge struct {
	graph.Edge
	id        string
	condition *graph.EdgeCondition
	from      graph.Node
	to        graph.Node
}

// NewEdge creates and returns a new edge with the specified from and to nodes.
func NewEdge(id string, from graph.Node, to graph.Node, condition *graph.EdgeCondition) graph.Edge {
	return &Edge{
		id:        id,
		condition: condition,
		from:      from,
		to:        to,
	}
}

// ID returns the edge ID
func (e *Edge) ID() string {
	return e.id
}

func (e *Edge) IsConditional() bool {
	return e.condition != nil
}

func (e *Edge) Condition() *graph.EdgeCondition {
	return e.condition
}

// From returns the edge from node
func (e *Edge) From() graph.Node {
	return e.from
}

// To returns the edge to node
func (e *Edge) To() graph.Node {
	return e.to
}
