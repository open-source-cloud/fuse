package workflow

type GraphNode struct {
	Node        *Node
	InputEdges  []*GraphEdge
	OutputEdges []*GraphEdge
}

type GraphEdge struct {
	Edge *Edge
	From *GraphNode
	To   *GraphNode
}

type Graph interface {
	Root() *GraphNode
	FindNode(id string) *GraphNode
	AddRootNode(node *GraphNode)
	AddNode(fromNodeId string, edge *GraphEdge, node *GraphNode)
}

type DefaultGraph struct {
	root *GraphNode
}

func NewDefaultGraph() *DefaultGraph {
	return &DefaultGraph{
		root: nil,
	}
}

func (g *DefaultGraph) Root() *GraphNode {
	return g.root
}

func (g *DefaultGraph) FindNode(id string) *GraphNode {
	var findNodeFunc func(node *GraphNode) *GraphNode
	findNodeFunc = func(node *GraphNode) *GraphNode {
		if node.Node.ID == id {
			return node
		}
		for _, edge := range node.OutputEdges {
			found := findNodeFunc(edge.To)
			if found != nil {
				return found
			}
		}
		return nil
	}
	return findNodeFunc(g.root)
}

func (g *DefaultGraph) AddRootNode(node *Node) {
	if g.root == nil {
		g.root = &GraphNode{
			Node:        node,
			InputEdges:  []*GraphEdge{},
			OutputEdges: []*GraphEdge{},
		}
	}
}

func (g *DefaultGraph) AddNode(fromNodeId string, edge *Edge, node *Node) {
	newNode := &GraphNode{
		Node:        node,
		InputEdges:  []*GraphEdge{},
		OutputEdges: []*GraphEdge{},
	}

	fromNode := g.FindNode(fromNodeId)
	if fromNode != nil {
		graphEdge := &GraphEdge{Edge: edge, From: fromNode, To: newNode}
		fromNode.OutputEdges = append(fromNode.OutputEdges, graphEdge)
		newNode.InputEdges = append(newNode.InputEdges, graphEdge)
	}
}
