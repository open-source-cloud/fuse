package graph

type Edge interface {
	From() Node
	To() Node
}

type edge struct {
	from Node
	to   Node
}

// NewEdge creates and returns a new edge with the specified from and to nodes.
func NewEdge(from Node, to Node) Edge {
	return &edge{
		from: from,
		to:   to,
	}
}

func (e *edge) From() Node {
	return e.from
}

func (e *edge) To() Node {
	return e.to
}
