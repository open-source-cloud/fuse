package workflow

import (
	"fmt"
)

type (
	// Node is a memory Graph Node
	Node struct {
		schema      *NodeSchema
		inputEdges  []*Edge
		outputEdges map[string]*Edge
	}
)

// newNode creates a new memory Graph Node
func newNode(schema *NodeSchema) *Node {
	return &Node{
		schema:      schema,
		inputEdges:  make([]*Edge, 0),
		outputEdges: make(map[string]*Edge),
	}
}

// ID returns the Node ID
func (n *Node) ID() string {
	return n.schema.ID
}

func (n *Node) FullID() string {
	return fmt.Sprintf("%s/%s", n.schema.Function, n.schema.ID)
}

func (n *Node) FunctionID() string {
	return n.schema.Function
}

func (n *Node) Schema() *NodeSchema {
	return n.schema
}

// InputEdges returns the input edges
func (n *Node) InputEdges() []*Edge {
	return n.inputEdges
}

// OutputEdges returns the output edges
func (n *Node) OutputEdges() map[string]*Edge {
	return n.outputEdges
}

// IsOutputConditional returns true if output is conditional, false otherwise
func (n *Node) IsOutputConditional() bool {
	return false
}

// AddInputEdge adds an input Edge
func (n *Node) AddInputEdge(edge *Edge) {
	n.inputEdges = append(n.inputEdges, edge)
}

// AddOutputEdge adds an output Edge
func (n *Node) AddOutputEdge(edgeID string, edge *Edge) {
	n.outputEdges[edgeID] = edge
}
