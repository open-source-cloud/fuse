package workflow

import (
	"fmt"
	"github.com/TyphonHill/go-mermaid/diagrams/flowchart"
	"github.com/open-source-cloud/fuse/pkg/arr"
	"gopkg.in/yaml.v3"
)

func NewGraphSchemaFromJSON(jsonSpec []byte) (*GraphSchema, error) {
	var schema GraphSchema
	err := yaml.Unmarshal(jsonSpec, &schema)
	if err != nil {
		return nil, err
	}
	return &schema, nil
}

func NewGraphSchemaFromYAML(yamlSpec []byte) (*GraphSchema, error) {
	var schema GraphSchema
	err := yaml.Unmarshal(yamlSpec, &schema)
	if err != nil {
		return nil, err
	}
	return &schema, nil
}

func NewGraphFromJSON(jsonSpec []byte) (*Graph, error) {
	schema, err := NewGraphSchemaFromJSON(jsonSpec)
	if err != nil {
		return nil, err
	}
	return NewGraphFromSchema(schema)
}

func NewGraphFromYAML(yamlSpec []byte) (*Graph, error) {
	schema, err := NewGraphSchemaFromYAML(yamlSpec)
	if err != nil {
		return nil, err
	}
	return NewGraphFromSchema(schema)
}

func NewGraphFromSchema(schema *GraphSchema) (*Graph, error) {
	graph := &Graph{
		schema: schema,
		nodes:  make(map[string]*Node),
		edges:  make(map[string]*Edge),
	}

	for i, nodeDef := range schema.Nodes {
		node := newNode(nodeDef)
		graph.nodes[node.ID()] = node
		if i == 0 {
			graph.trigger = node
		}
	}
	for _, edgeDef := range schema.Edges {
		fromNode, err := graph.FindNode(edgeDef.From)
		if err != nil {
			return nil, err
		}
		toNode, err := graph.FindNode(edgeDef.To)
		if err != nil {
			return nil, err
		}
		edge := newEdge(edgeDef.ID, fromNode, toNode, edgeDef)
		graph.edges[edge.ID()] = edge
		fromNode.AddOutputEdge(edge)
		toNode.AddInputEdge(edge)
	}

	graph.calculateThreads()

	return graph, nil
}

type (
	Graph struct {
		trigger *Node
		schema  *GraphSchema
		nodes   map[string]*Node
		edges   map[string]*Edge
	}
)

func (g *Graph) ID() string {
	return g.schema.ID
}

// Trigger returns the root Node of the Graph
func (g *Graph) Trigger() *Node {
	return g.trigger
}

// FindNode finds a Node by ID
func (g *Graph) FindNode(nodeID string) (*Node, error) {
	if g.trigger.ID() == nodeID {
		return g.trigger, nil
	}

	if nodeRef, ok := g.nodes[nodeID]; ok {
		return nodeRef, nil
	}

	return nil, fmt.Errorf("node %s not found", nodeID)
}

func (g *Graph) MermaidFlowchart() string {
	chart := flowchart.NewFlowchart()
	nodeKeys := make(map[*Node]*flowchart.Node)

	// Add nodes with custom labels
	for _, node := range g.nodes {
		var label string
		if len(node.InputEdges()) > 0 && node.InputEdges()[0].IsConditional() {
			label = fmt.Sprintf("ID: %s\\nFunction: %s\\nThread: %d\\n(cond: %s)",
				node.ID(), node.FunctionID(), node.Thread(), node.InputEdges()[0].Condition().Name)
		} else {
			label = fmt.Sprintf("ID: %s\\nFunction: %s\\nThread: %d",
				node.ID(), node.FunctionID(), node.Thread())
		}
		key := chart.AddNode(label)
		nodeKeys[node] = key
	}

	// Add links (edges) between nodes
	for _, node := range g.nodes {
		src := nodeKeys[node]
		for _, edge := range node.OutputEdges() {
			dst := nodeKeys[edge.To()]
			chart.AddLink(src, dst)
		}
	}

	// Render result
	return chart.String()

}

func (g *Graph) calculateThreads() {
	threads := map[int][]*Node{}

	var dfs func(*Node, int)
	dfs = func(node *Node, thread int) {
		if len(threads) <= thread {
			threads[thread] = []*Node{}
		}
		threads[thread] = append(threads[thread], node)
		node.thread = thread

		// Split edges by conditional, if they exist. Use "" for non-conditional edges
		conditionalEdges := map[string][]*Edge{"": []*Edge{}}
		for _, edge := range node.OutputEdges() {
			if edge.IsConditional() {
				conditionalEdges[edge.Condition().Name] = append(conditionalEdges[edge.Condition().Name], edge)
			} else {
				conditionalEdges[""] = append(conditionalEdges[""], edge)
			}
		}
		// walk through all edges, by conditional
		for _, edges := range conditionalEdges {
			for i, edge := range edges {
				newThread := thread + i
				for i, threadList := range threads {
					if arr.Contains(threadList, edge.To()) {
						newThread = i
					}
				}
				dfs(edge.To(), newThread)
			}
		}
	}
	dfs(g.trigger, 0)
}
