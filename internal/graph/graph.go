// Package graph provides a memory graph implementation
package graph

import (
	"fmt"

	"github.com/open-source-cloud/fuse/pkg/graph"
)

// Graph is a memory graph
type Graph struct {
	root graph.Node
}

// NewGraph creates a new Graph with a root node
func NewGraph(root graph.Node) *Graph {
	return &Graph{
		root: root,
	}
}

// Root returns the root node of the graph
func (g *Graph) Root() graph.Node {
	return g.root
}

// FindNode finds a node by ID
func (g *Graph) FindNode(nodeID string) (graph.Node, error) {
	var find func(node graph.Node) graph.Node
	find = func(node graph.Node) graph.Node {
		if node.ID() == nodeID {
			return node
		}
		for _, edge := range node.OutputEdges() {
			result := find(edge.To())
			if result != nil {
				return result
			}
		}
		return nil
	}
	node := find(g.root)
	if node == nil {
		return nil, fmt.Errorf("node %s not found", nodeID)
	}
	return node, nil
}

// AddNode adds a node to the graph
func (g *Graph) AddNode(parentNodeID string, edgeID string, node graph.Node, condition *graph.EdgeCondition) error {
	parentNode, err := g.FindNode(parentNodeID)
	if err != nil {
		return err
	}
	newEdge := NewEdge(edgeID, parentNode, node, condition)
	parentNode.AddOutputEdge(edgeID, newEdge)
	node.AddInputEdge(newEdge)
	return nil
}

// AddNodeMultipleParents adds a node to the graph with multiple parents
func (g *Graph) AddNodeMultipleParents(parentNodeIDs []graph.ParentNodeWithCondition, edgeID string, node graph.Node) error {
	for _, parent := range parentNodeIDs {
		err := g.AddNode(parent.NodeID, edgeID, node, parent.Condition)
		if err != nil {
			return err
		}
	}
	return nil
}
