package graph

import (
	"fmt"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

type NodeInputMapping struct {
	Source    string
	ParamName string
	Mapping   string
}

type NodeConfig interface {
	InputMapping() []NodeInputMapping
	AddInputMapping(source string, paramName string, mapping string)
}

type nodeConfig struct {
	inputMapping []NodeInputMapping
}

func NewNodeConfig() NodeConfig {
	return &nodeConfig{
		inputMapping: []NodeInputMapping{},
	}
}

func (c *nodeConfig) InputMapping() []NodeInputMapping {
	return c.inputMapping
}

func (c *nodeConfig) AddInputMapping(source string, paramName string, mapping string) {
	c.inputMapping = append(c.inputMapping, NodeInputMapping{
		Source:    source,
		ParamName: paramName,
		Mapping:   mapping,
	})
}

type Node interface {
	ID() string
	NodeRef() workflow.Node
	Config() NodeConfig
	InputEdges() []Edge
	OutputEdges() map[string]Edge
	AddInputEdge(edge Edge)
	AddOutputEdge(edgeId string, edge Edge)
}

type node struct {
	id          string
	node        workflow.Node
	config      NodeConfig
	inputEdges  []Edge
	outputEdges map[string]Edge
}

// NewNode creates a new instance of node with the given parameters.
func NewNode(uuid string, workflowNode workflow.Node, config NodeConfig) Node {
	return &node{
		id:          fmt.Sprintf("%s/%s", workflowNode.ID(), uuid),
		node:        workflowNode,
		config:      config,
		inputEdges:  []Edge{},
		outputEdges: make(map[string]Edge),
	}
}

func (n *node) ID() string {
	return n.id
}

func (n *node) NodeRef() workflow.Node {
	return n.node
}

func (n *node) Config() NodeConfig {
	return n.config
}

func (n *node) InputEdges() []Edge {
	return n.inputEdges
}

func (n *node) OutputEdges() map[string]Edge {
	return n.outputEdges
}

func (n *node) AddInputEdge(edge Edge) {
	n.inputEdges = append(n.inputEdges, edge)
}

func (n *node) AddOutputEdge(edgeId string, edge Edge) {
	n.outputEdges[edgeId] = edge
}
