package graph

import "github.com/open-source-cloud/fuse/pkg/workflow"

type Edge interface {
	ID() string
	EdgeRef() workflow.Edge
	From() Node
	To() Node
}

type SimpleEdge struct {
	id      string
	edgeRef workflow.Edge
	from    Node
	to      Node
}

// NewSimpleEdge creates a new instance of SimpleEdge.
func NewSimpleEdge(id string, edgeRef workflow.Edge, from Node, to Node) *SimpleEdge {
	return &SimpleEdge{
		id:      id,
		edgeRef: edgeRef,
		from:    from,
		to:      to,
	}
}

func (e *SimpleEdge) ID() string {
	return e.id
}

func (e *SimpleEdge) EdgeRef() workflow.Edge {
	return e.edgeRef
}

func (e *SimpleEdge) From() Node {
	return e.from
}

func (e *SimpleEdge) To() Node {
	return e.to
}
