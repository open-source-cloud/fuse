// Package graph provides the graph interfaces for workflows
package graph

// Edge describes a graph's edge
type Edge interface {
	ID() string
	From() Node
	To() Node
}

type edge struct {
	id   string
	from Node
	to   Node
}

// NewEdge creates and returns a new edge with the specified from and to nodes.
func NewEdge(id string, from Node, to Node) Edge {
	return &edge{
		id:   id,
		from: from,
		to:   to,
	}
}

func (e *edge) ID() string {
	return e.id
}

func (e *edge) From() Node {
	return e.from
}

func (e *edge) To() Node {
	return e.to
}
