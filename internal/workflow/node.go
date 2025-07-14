package workflow

import (
	"fmt"
	"github.com/open-source-cloud/fuse/internal/packages"
)

type (
	// Node is a Graph Node
	Node struct {
		schema           *NodeSchema
		functionMetadata *packages.FunctionMetadata
		thread           uint16
		parentThreads    []uint16
		inputEdges       []*Edge
		outputEdges      []*Edge
	}
)

// newNode creates a new Graph Node
func newNode(schema *NodeSchema) *Node {
	return &Node{
		schema:        schema,
		thread:        0,
		parentThreads: []uint16{},
		inputEdges:    make([]*Edge, 0),
		outputEdges:   make([]*Edge, 0),
	}
}

// ID returns the Node ID
func (n *Node) ID() string {
	return n.schema.ID
}

// FullID returns the full ID for the Node (that is namespaces by the Function ID)
func (n *Node) FullID() string {
	return fmt.Sprintf("%s/%s", n.schema.Function, n.schema.ID)
}

// FunctionID function ID
func (n *Node) FunctionID() string {
	return n.schema.Function
}

// Schema the schema that represents this Node
func (n *Node) Schema() *NodeSchema {
	return n.schema
}

// Thread the Thread ID for this node
func (n *Node) Thread() uint16 {
	return n.thread
}

// FunctionMetadata function metadata for this Node
func (n *Node) FunctionMetadata() *packages.FunctionMetadata {
	return n.functionMetadata
}

// InputEdges returns the input edges
func (n *Node) InputEdges() []*Edge {
	return n.inputEdges
}

// OutputEdges returns the output edges
func (n *Node) OutputEdges() []*Edge {
	return n.outputEdges
}

// IsConditional returns true if this node has conditional output
func (n *Node) IsConditional() bool {
	return n.functionMetadata.Output.ConditionalOutput
}

// AddInputEdge adds an input Edge
func (n *Node) AddInputEdge(edge *Edge) {
	n.inputEdges = append(n.inputEdges, edge)
}

// AddOutputEdge adds an output Edge
func (n *Node) AddOutputEdge(edge *Edge) {
	n.outputEdges = append(n.outputEdges, edge)
}
