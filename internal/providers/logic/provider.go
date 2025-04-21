// Package logic provides logic nodes for workflows
package logic

import (
	"fmt"

	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// LogicProviderID is the ID of the logic node provider
const LogicProviderID = "fuse.io/workflows/internal/logic"

// NodeProvider is a logic node provider
type NodeProvider struct {
	workflow.NodeProvider
	id    string
	nodes map[string]workflow.Node
}

// NewNodeProvider creates a new logic node provider
func NewNodeProvider() workflow.NodeProvider {
	nodeProvider := &NodeProvider{
		id:    LogicProviderID,
		nodes: make(map[string]workflow.Node),
	}

	nodeProvider.nodes[SumNodeID] = NewSumNode()
	nodeProvider.nodes[RandNodeID] = NewRandNode()

	return nodeProvider
}

// ID returns the ID of the NodeProvider
func (np *NodeProvider) ID() string {
	return np.id
}

// Nodes returns all nodes in the provider
func (np *NodeProvider) Nodes() []workflow.Node {
	values := make([]workflow.Node, 0, len(np.nodes))
	for _, node := range np.nodes {
		values = append(values, node)
	}
	return values
}

// GetNode returns a node by ID
func (np *NodeProvider) GetNode(id string) (workflow.Node, error) {
	node, ok := np.nodes[id]
	if !ok {
		return nil, fmt.Errorf("node %s not found", id)
	}
	return node, nil
}
