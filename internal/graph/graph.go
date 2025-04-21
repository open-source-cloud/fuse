// Package graph provides the graph interfaces for workflows
package graph

// Graph describes a graph interface
type Graph interface {
	Root() Node
	FindNode(nodeID string) Node
	AddNode(parentNodeID string, edgeID string, node Node)
	AddNodeMultipleParents(parentNodeIDs []string, edgeID string, node Node)
}

type graph struct {
	root Node
}

// NewGraph creates a new Graph with a root node
func NewGraph(root Node) Graph {
	return &graph{
		root: root,
	}
}

func (g *graph) Root() Node {
	return g.root
}

func (g *graph) FindNode(nodeID string) Node {
	var find func(node Node) Node
	find = func(node Node) Node {
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

func (g *graph) AddNode(parentNodeID string, edgeID string, node Node) {
	parentNode := g.FindNode(parentNodeID)
	newEdge := NewEdge(edgeID, parentNode, node)
	parentNode.AddOutputEdge(edgeID, newEdge)
	node.AddInputEdge(newEdge)
}

func (g *graph) AddNodeMultipleParents(parentNodeIDs []string, edgeID string, node Node) {
	for _, parentNodeID := range parentNodeIDs {
		g.AddNode(parentNodeID, edgeID, node)
	}
}
