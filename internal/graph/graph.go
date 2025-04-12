package graph

type Graph interface {
	Root() Node
	FindNode(nodeId string) Node
	AddNode(parentNodeId string, edgeIdentifier string, node Node)
}

type graph struct {
	root Node
}

func NewGraph(root Node) Graph {
	return &graph{
		root: root,
	}
}

func (g *graph) Root() Node {
	return g.root
}

func (g *graph) FindNode(nodeId string) Node {
	var find func(node Node) Node
	find = func(node Node) Node {
		if node.ID() == nodeId {
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

func (g *graph) AddNode(parentNodeId string, edgeIdentifier string, node Node) {
	parentNode := g.FindNode(parentNodeId)
	parentNode.AddOutputEdge(edgeIdentifier, NewEdge(parentNode, node))
}
