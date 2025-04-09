package graph

import (
	"fmt"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

type Node interface {
	ID() string
	NodeRef() workflow.Node
	InputEdges() []Edge
	OutputEdges() []Edge
}

type node struct {
	id          string
	node        workflow.Node
	inputEdges  []Edge
	outputEdges []Edge
}

// NewNode creates a new instance of node with the given parameters.
func NewNode(uuid string, workflowNode workflow.Node) Node {
	return &node{
		id:          fmt.Sprintf("%s/%s", workflowNode.ID(), uuid),
		node:        workflowNode,
		inputEdges:  []Edge{},
		outputEdges: []Edge{},
	}
}

func (n *node) ID() string {
	return n.id
}

func (n *node) NodeRef() workflow.Node {
	return n.node
}

func (n *node) InputEdges() []Edge {
	return n.inputEdges
}

func (n *node) OutputEdges() []Edge {
	return n.outputEdges
}
