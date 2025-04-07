package graph

import (
	"fmt"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

type Graph interface {
	Root() Node
	FindNode(id string) (Node, error)
	FindEdge(id string) (Edge, error)
	AddNode(previousNodeId string, edgeId string, edge workflow.Edge, nodeId string, node workflow.Node) error
}

type SimpleGraph struct {
	root    Node
	nodeMap map[string]Node
	edgeMap map[string]Edge
}

func NewGraph(rootId string, root workflow.Node) *SimpleGraph {
	return &SimpleGraph{
		root:    NewSimpleNode(rootId, root),
		nodeMap: make(map[string]Node),
		edgeMap: make(map[string]Edge),
	}
}

func (n *SimpleGraph) Root() Node {
	return n.root
}

func (n *SimpleGraph) FindNode(id string) (Node, error) {
	if node, ok := n.nodeMap[id]; ok {
		return node, nil
	}
	return nil, fmt.Errorf("node %s not found", id)
}

func (n *SimpleGraph) FindEdge(id string) (Edge, error) {
	if edge, ok := n.edgeMap[id]; ok {
		return edge, nil
	}
	return nil, fmt.Errorf("edge %s not found", id)
}

func (n *SimpleGraph) AddNode(previousNodeId string, edgeId string, edge workflow.Edge, nodeId string, node workflow.Node) error {
	if _, exists := n.nodeMap[nodeId]; exists {
		return fmt.Errorf("node %s already exists", nodeId)
	}
	if _, exists := n.edgeMap[edgeId]; exists {
		return fmt.Errorf("edge %s already exists", edgeId)
	}

	previousNode, exists := n.nodeMap[previousNodeId]
	if !exists {
		return fmt.Errorf("previous node %s does not exist", previousNodeId)
	}

	newNode := NewSimpleNode(nodeId, node)
	newEdge := NewSimpleEdge(edgeId, edge, n.nodeMap[previousNodeId], newNode)

	// Add the edge to the previous node
	previousNode.AddOutputEdge(newEdge)
	newNode.AddInputEdge(newEdge)

	// add to map for quicker referencing
	n.nodeMap[nodeId] = newNode
	n.edgeMap[edgeId] = newEdge

	return nil
}
