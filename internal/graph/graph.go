// Package graph provides a memory graph implementation
package graph

import (
	"fmt"
	"github.com/open-source-cloud/fuse/internal/graph/schema"
)

type (
	// ParentNodeWithCondition represents a parent node with a condition to be added
	ParentNodeWithCondition struct {
		NodeID    string
		Condition *schema.EdgeCondition
		input []schema.InputMapping
	}
	// Graph is the interface for a graph
	Graph interface {
		Root() Node
		FindNode(nodeID string) (Node, error)
		AddNode(parentNodeID string, edgeID string, node Node, condition *schema.EdgeCondition, input []schema.InputMapping) error
		AddNodeMultipleParents(parentNodeIDs []ParentNodeWithCondition, edgeID string, node Node) error
	}

	graph struct {
		root  Node
		nodes map[string]Node
		edges map[string]Edge
	}
)

// NewGraph creates a new Graph with a root node
func NewGraph(root Node) Graph {
	return &graph{
		root: root,
	}
}

// Root returns the root node of the graph
func (g *graph) Root() Node {
	return g.root
}

// FindNode finds a node by ID
func (g *graph) FindNode(nodeID string) (Node, error) {
	if g.root.ID() == nodeID {
		return g.root, nil
	}

	if node, ok := g.nodes[nodeID]; ok {
		return node, nil
	}

	return nil, fmt.Errorf("node %s not found", nodeID)
}

// AddNode adds a node to the graph
func (g *graph) AddNode(parentNodeID string, edgeID string, node Node, condition *schema.EdgeCondition, input []schema.InputMapping) error {
	parentNode, err := g.FindNode(parentNodeID)
	if err != nil {
		return err
	}
	newEdge := NewEdge(edgeID, parentNode, node, condition, input)
	parentNode.AddOutputEdge(edgeID, newEdge)
	node.AddInputEdge(newEdge)
	return nil
}

// AddNodeMultipleParents adds a node to the graph with multiple parents
func (g *graph) AddNodeMultipleParents(parentNodeIDs []ParentNodeWithCondition, edgeID string, node Node) error {
	for _, parent := range parentNodeIDs {
		err := g.AddNode(parent.NodeID, edgeID, node, parent.Condition, parent.input)
		if err != nil {
			return err
		}
	}
	return nil
}
