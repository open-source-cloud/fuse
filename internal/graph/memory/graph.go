// Package memory provides a memory graph implementation
package memory

import "github.com/open-source-cloud/fuse/pkg/graph"

// Graph is a memory graph
type Graph struct {
	root graph.Node
}

// NewGraph creates a new Graph with a root node
func NewGraph(root graph.Node) graph.Graph {
	return &Graph{
		root: root,
	}
}

// Root returns the root node of the graph
func (g *Graph) Root() graph.Node {
	return g.root
}

// FindNode finds a node by ID
func (g *Graph) FindNode(nodeID string) graph.Node {
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
	return find(g.root)
}

// AddNode adds a node to the graph
func (g *Graph) AddNode(parentNodeID string, edgeID string, node graph.Node) {
	parentNode := g.FindNode(parentNodeID)
	newEdge := NewEdge(edgeID, parentNode, node)
	parentNode.AddOutputEdge(edgeID, newEdge)
	node.AddInputEdge(newEdge)
}

// AddNodeMultipleParents adds a node to the graph with multiple parents
func (g *Graph) AddNodeMultipleParents(parentNodeIDs []string, edgeID string, node graph.Node) {
	for _, parentNodeID := range parentNodeIDs {
		g.AddNode(parentNodeID, edgeID, node)
	}
}
