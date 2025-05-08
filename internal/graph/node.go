// Package graph provides a memory graph implementation
package graph

import (
	"fmt"
	"github.com/open-source-cloud/fuse/internal/graph/schema"
)

type (
	// Node describes an executable Node object
	Node interface {
		ID() string
		FullID() string
		Schema() *schema.Node
		InputEdges() []Edge
		OutputEdges() map[string]Edge
		IsOutputConditional() bool
		AddInputEdge(edge Edge)
		AddOutputEdge(edgeID string, edge Edge)
	}

	// Node is a memory graph node
	node struct {
		schema      *schema.Node
		inputEdges  []Edge
		outputEdges map[string]Edge
	}
)

// NewNode creates a new memory graph node
func NewNode(schemaDef *schema.Node) Node {
	return &node{
		schema:      schemaDef,
		inputEdges:  make([]Edge, 0),
		outputEdges: make(map[string]Edge),
	}
}

// ID returns the node ID
func (n *node) ID() string {
	return n.schema.ID
}

func (n *node) FullID() string {
	return fmt.Sprintf("%s/%s", n.schema.Function, n.schema.ID)
}

func (n *node) Schema() *schema.Node {
	return n.schema
}

// InputEdges returns the input edges
func (n *node) InputEdges() []Edge {
	return n.inputEdges
}

// OutputEdges returns the output edges
func (n *node) OutputEdges() map[string]Edge {
	return n.outputEdges
}

// IsOutputConditional returns true if output is conditional, false otherwise
func (n *node) IsOutputConditional() bool {
	return false
}

// AddInputEdge adds an input edge
func (n *node) AddInputEdge(edge Edge) {
	n.inputEdges = append(n.inputEdges, edge)
}

// AddOutputEdge adds an output edge
func (n *node) AddOutputEdge(edgeID string, edge Edge) {
	n.outputEdges[edgeID] = edge
}
