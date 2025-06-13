package workflow
// Package packages workflow function packages

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
	// Graph defines a Workflow Graph
	Graph struct {
		trigger *Node
		schema  *GraphSchema
		nodes   map[string]*Node
		edges   map[string]*Edge
	}
)

// ID the schema ID for the graph
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

// MermaidFlowchart generates a Mermaid flowchart
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

// calculateThreads assigns thread IDs and parentThreads for nodes in the workflow graph.
//
// It handles forks (where a node has multiple outgoing edges) by assigning new threads to each branch. It
//  handles joins (where a node has multiple incoming edges) by collecting unique parent thread IDs and
// issuing a new thread for the merged path. Cycles/loops are supported, and parentThreads are stabilized
// in a second pass to avoid capturing "ghost" threads that never actually contribute a live path.
//
// This process has two phases:
//
// Phase 1: Thread Assignment
// - Uses a depth-first walk to explore all possible execution threads, forking and joining as appropriate.
// - Assigns thread IDs to nodes as each path is discovered. At joins, a new thread is issued and "guessed" parentThreads set.
//
// Phase 2: Parent Thread Stabilization
//   - Iterates over all join nodes AFTER threads have propagated, collecting stable and actual parent thread IDs
//     (and filtering out any "ghosts" arising from cycles or unstable recursive order).
//   - Sets final, accurate parentThreads for each join.
//
// This ensures:
// - Every thread value is unique, local to a branch.
// - Joins always reference actual upstream parent threads.
// - Cycles/loops do not cause recursion overflows or ghost dependencies.
func (g *Graph) calculateThreads() {
	// Visited tracks if a node+thread+parentThreads combination has already been explored.
	// Prevents redundant work and handles graphs with cycles.
	visited := make(map[string]map[string]bool) // node.ID() -> visit key -> bool

	var threadCounter uint16 // Used to issue new, unique thread IDs as needed

	// Helper to create a new, unique thread ID each time it's called.
	newThreadID := func() uint16 {
		threadCounter++
		return threadCounter
	}

	// Helper to convert a parentThreads slice into a canonical (sorted, comma-separated) string.
	// This is used for visit deduplication, to ensure that different orderings of the same set are treated as equal.
	makeParentKey := func(parentThreads []uint16) string {
		if len(parentThreads) == 0 {
			return ""
		}
		cp := append([]uint16(nil), parentThreads...) // defensive copy
		sort.Slice(cp, func(i, j int) bool {
			return cp[i] < cp[j]
		})
		key := ""
		for i, t := range cp {
			if i > 0 {
				key += ","
			}
			key += fmt.Sprintf("%d", t)
		}
		return key
	}

	// PHASE 1: Assign threads by walking through the graph
	// The walk function explores all possible paths from the root (trigger), assigning threads and "guessing" parentThreads at joins.
	var walk func(node *Node, thread uint16, parentThreads []uint16, inPath map[string]bool)
	walk = func(node *Node, thread uint16, parentThreads []uint16, inPath map[string]bool) {
		id := node.ID()
		visitKey := fmt.Sprintf("%d|%s", thread, makeParentKey(parentThreads))

		// Only proceed if this node+thread+parentThreads combination hasn't already been covered.
		if visited[id] == nil {
			visited[id] = map[string]bool{}
		}
		if visited[id][visitKey] {
			return
		}
		visited[id][visitKey] = true

		// Detect cycles: if this node has already been visited in the current call stack, return to prevent infinite recursion.
		if inPath[id] {
			return
		}
		inPath[id] = true

		// Set this node's thread for this traversal path
		node.thread = thread

		// Store parentThreads for documentation; will be stabilized in phase 2
		if len(parentThreads) > 1 {
			// Defensive copy
			node.parentThreads = append([]uint16(nil), parentThreads...)
		} else {
			node.parentThreads = nil
		}

		// Join handling: this node has multiple incoming edges (i.e., is a "join" node)
		if len(node.InputEdges()) > 1 {
			// Gather thread IDs from all immediate parent nodes (these may not be final, hence "guessing")
			parentThreadSet := make(map[uint16]struct{})
			for _, edge := range node.InputEdges() {
				pNode := edge.From()
				pThread := pNode.thread
				// The thread on the trigger node is 0, which is valid
				if pThread != 0 || pNode == g.trigger {
					parentThreadSet[pThread] = struct{}{}
				}
			}
			// If more than one parent thread reaches this node, it's an actual join; issue a new thread, and record parentThreads
			if len(parentThreadSet) > 1 {
				pt := make([]uint16, 0, len(parentThreadSet))
				for p := range parentThreadSet {
					pt = append(pt, p)
				}
				sort.Slice(pt, func(i, j int) bool {
					return pt[i] < pt[j]
				})
				newThread := newThreadID()
				node.thread = newThread
				node.parentThreads = pt
				thread = newThread // carry this updated thread downstream
			}
		}

		// Handle forks (multiple outgoing edges for the same condition—i.e., parallel branches or conditional branches)
		// Group outgoing edges by their condition label
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
				// Simple outgoing edge, continue the current thread
				walk(edges[0].To(), thread, nil, inPath)
			} else if len(edges) > 1 {
				// Fork: each branch gets a new thread, inheriting from this node
				for _, edge := range edges {
					tID := newThreadID()
					walk(edge.To(), tID, nil, inPath)
				}
			}
		}
		// Finished exploring from this node in the current traversal, pop from inPath
		inPath[id] = false
	}

	// Begin traversal from the trigger/root node, assigning thread 0.
	walk(g.trigger, 0, []uint16{0}, map[string]bool{})

	// PHASE 2: Stabilize parentThreads at all join nodes
	// For every join node (node with >1 input edge), recompute parentThreads from finalized parent.node.thread values
	// This ensures that cycles can't create "ghost" thread IDs.
	for _, node := range g.nodes {
		if len(node.InputEdges()) > 1 {
			parentThreadSet := make(map[uint16]struct{})
			for _, edge := range node.InputEdges() {
				pNode := edge.From()
				if pNode != nil {
					parentThreadSet[pNode.thread] = struct{}{}
				}
			}
			// Remove our own thread ID (in self-cycle/cross-loop cases) to avoid tautological dependency
			delete(parentThreadSet, node.thread)
			if len(parentThreadSet) > 1 {
				// Sort for consistent/stable output
				pt := make([]uint16, 0, len(parentThreadSet))
				for p := range parentThreadSet {
					pt = append(pt, p)
				}
				sort.Slice(pt, func(i, j int) bool {
					return pt[i] < pt[j]
				})
				node.parentThreads = pt
			} else {
				// No true join; clear
				node.parentThreads = nil
			}
		} else {
			// This is not a join; clear any previously assigned parentThreads
			node.parentThreads = nil
		}
	}
}
