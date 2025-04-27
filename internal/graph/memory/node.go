// Package memory provides a memory graph implementation
package memory

import (
	"fmt"

	"github.com/open-source-cloud/fuse/pkg/graph"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

type (
	// Node is a memory graph node
	Node struct {
		graph.Node
		id           string
		workflowNode workflow.Node
		config       graph.NodeConfig
		inputEdges   []graph.Edge
		outputEdges  map[string]graph.Edge
	}
	// NodeConfig is a memory graph node config
	NodeConfig struct {
		graph.NodeConfig
		inputMapping []graph.NodeInputMapping
	}
)

// NewNode creates a new memory graph node
func NewNode(uuid string, workflowNode workflow.Node, config graph.NodeConfig) *Node {
	return &Node{
		id:           fmt.Sprintf("%s/%s", workflowNode.ID(), uuid),
		workflowNode: workflowNode,
		config:       config,
		inputEdges:   make([]graph.Edge, 0),
		outputEdges:  make(map[string]graph.Edge),
	}
}

// ID returns the node ID
func (n *Node) ID() string {
	return n.id
}

// NodeRef returns the node reference
func (n *Node) NodeRef() workflow.Node {
	return n.workflowNode
}

// Config returns the node config
func (n *Node) Config() graph.NodeConfig {
	return n.config
}

// InputEdges returns the input edges
func (n *Node) InputEdges() []graph.Edge {
	return n.inputEdges
}

// OutputEdges returns the output edges
func (n *Node) OutputEdges() map[string]graph.Edge {
	return n.outputEdges
}

// AddInputEdge adds an input edge
func (n *Node) AddInputEdge(edge graph.Edge) {
	n.inputEdges = append(n.inputEdges, edge)
}

// AddOutputEdge adds an output edge
func (n *Node) AddOutputEdge(edgeID string, edge graph.Edge) {
	n.outputEdges[edgeID] = edge
}

// NewNodeConfig creates a new memory graph node config
func NewNodeConfig() *NodeConfig {
	return &NodeConfig{
		inputMapping: make([]graph.NodeInputMapping, 0),
	}
}

// InputMapping returns the input mapping
func (c *NodeConfig) InputMapping() []graph.NodeInputMapping {
	return c.inputMapping
}

// AddInputMapping adds an input mapping
func (c *NodeConfig) AddInputMapping(source string, origin string, mapping string) {
	c.inputMapping = append(c.inputMapping, graph.NodeInputMapping{
		Source:  source,
		Origin:  origin,
		Mapping: mapping,
	})
}
