package workflow

import (
	"fmt"
	"github.com/TyphonHill/go-mermaid/diagrams/flowchart"
	"gopkg.in/yaml.v3"
	"sort"
)

func newGraphSchemaFromJSON(jsonSpec []byte) (*GraphSchema, error) {
	var schema GraphSchema
	err := yaml.Unmarshal(jsonSpec, &schema)
	if err != nil {
		return nil, err
	}
	return &schema, nil
}

func newGraphSchemaFromYAML(yamlSpec []byte) (*GraphSchema, error) {
	var schema GraphSchema
	err := yaml.Unmarshal(yamlSpec, &schema)
	if err != nil {
		return nil, err
	}
	return &schema, nil
}

func newGraphFromJSON(jsonSpec []byte) (*Graph, error) {
	schema, err := newGraphSchemaFromJSON(jsonSpec)
	if err != nil {
		return nil, err
	}
	return newGraphFromSchema(schema)
}

func newGraphFromYAML(yamlSpec []byte) (*Graph, error) {
	schema, err := newGraphSchemaFromYAML(yamlSpec)
	if err != nil {
		return nil, err
	}
	return newGraphFromSchema(schema)
}

func newGraphFromSchema(schema *GraphSchema) (*Graph, error) {
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
	chart.Config.SetDiagramPadding(30)
	nodeKeys := make(map[*Node]*flowchart.Node)

	// set trigger/start node
	labelTrigger := fmt.Sprintf("id: %s\\nFunction: %s\\nThread: %d",
		g.trigger.ID(), g.trigger.FunctionID(), g.trigger.thread)
	trigger := chart.AddNode(labelTrigger)
	trigger.Shape = flowchart.NodeShapeStart
	nodeKeys[g.trigger] = trigger

	// Add nodes with custom labels
	for _, node := range g.nodes {
		if node == g.trigger {
			continue
		}
		label := fmt.Sprintf("id: %s\\nFunction: %s\\nThread: %d\\nParentThreads: %v",
			node.ID(), node.FunctionID(), node.Thread(), node.parentThreads)
		if len(node.InputEdges()) > 0 && node.InputEdges()[0].IsConditional() {
			label = fmt.Sprintf("%s\\n(cond: %s)", label, node.InputEdges()[0].Condition().Name)
		}
		key := chart.AddNode(label)
		for _, edge := range node.OutputEdges() {
			if edge.IsConditional() {
				key.Shape = flowchart.NodeShapeDecision
				break
			}
		}
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
	visited := make(map[string]map[string]bool) // node.id() -> id-key -> bool
	var threadCounter int

	newThreadID := func() int {
		threadCounter++
		return threadCounter
	}

	// Helper to create a stable string from a list of parent threads
	makeParentKey := func(parentThreads []int) string {
		if len(parentThreads) == 0 {
			return ""
		}
		cp := append([]int(nil), parentThreads...)
		sort.Ints(cp)
		key := ""
		for i, t := range cp {
			if i > 0 {
				key += ","
			}
			key += fmt.Sprintf("%d", t)
		}
		return key
	}

	var walk func(node *Node, thread int, parentThreads []int)
	walk = func(node *Node, thread int, parentThreads []int) {
		key := fmt.Sprintf("%d|%s", thread, makeParentKey(parentThreads))
		id := node.ID()
		if visited[id] == nil {
			visited[id] = map[string]bool{}
		}
		if visited[id][key] {
			return
		}
		visited[id][key] = true

		node.thread = thread

		// At a join, use the true parent set; else, don't set self as parent
		if len(parentThreads) > 1 {
			node.parentThreads = append([]int(nil), parentThreads...) // These are real merging parents!
		} else {
			node.parentThreads = nil // No multi-parent, so leave empty or nil
		}

		wasJoin := false
		// Handle join/merge: multiple incoming edges
		if len(node.InputEdges()) > 1 {
			parentThreadSet := make(map[int]struct{})
			for _, edge := range node.InputEdges() {
				p := edge.From().thread
				parentThreadSet[p] = struct{}{}
			}
			if len(parentThreadSet) > 1 {
				pt := make([]int, 0, len(parentThreadSet))
				for p := range parentThreadSet {
					pt = append(pt, p)
				}
				sort.Ints(pt)
				newThread := newThreadID()
				node.thread = newThread
				node.parentThreads = pt // True parents
				thread = newThread
				//parentThreads = pt // Propagate for downstream, but was a join
				wasJoin = true
			}
		}

		// Group output edges by condition name
		condGroups := make(map[string][]*Edge)
		for _, edge := range node.OutputEdges() {
			condName := ""
			if edge.Condition() != nil {
				condName = edge.Condition().Name
			}
			condGroups[condName] = append(condGroups[condName], edge)
		}

		for _, edges := range condGroups {
			if len(edges) == 1 {
				// If this node was a join, downstream becomes single-parent (this id)
				if wasJoin {
					walk(edges[0].To(), thread, nil)
				} else {
					walk(edges[0].To(), thread, nil)
				}
			} else if len(edges) > 1 {
				// Fork: each outgoing edge is a new id, parent is the fork point's id
				for _, edge := range edges {
					tID := newThreadID()
					walk(edge.To(), tID, nil)
				}
			}
		}
	}

	walk(g.trigger, 0, []int{0})

}
