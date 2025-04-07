package graph

import "github.com/open-source-cloud/fuse/pkg/workflow"

type Node interface {
	ID() string
	NodeRef() workflow.Node
	InputEdges() []Edge
	OutputEdges() []Edge
	AddInputEdge(edge Edge)
	AddOutputEdge(edge Edge)
}

type SimpleNode struct {
	id          string
	nodeRef     workflow.Node
	inputEdges  []Edge
	outputEdges []Edge
}

func NewSimpleNode(id string, node workflow.Node) *SimpleNode {
	// Construct a SimpleNode and set initial values if needed
	return &SimpleNode{
		id:          id,
		nodeRef:     node,
		inputEdges:  []Edge{},
		outputEdges: []Edge{},
	}
}

func (n *SimpleNode) ID() string {
	return n.id
}

func (n *SimpleNode) NodeRef() workflow.Node {
	return n.nodeRef
}

func (n *SimpleNode) InputEdges() []Edge {
	return n.inputEdges
}

func (n *SimpleNode) OutputEdges() []Edge {
	return n.outputEdges
}

func (n *SimpleNode) AddInputEdge(edge Edge) {
	n.inputEdges = append(n.inputEdges, edge)
}

func (n *SimpleNode) AddOutputEdge(edge Edge) {
	n.outputEdges = append(n.outputEdges, edge)
}
