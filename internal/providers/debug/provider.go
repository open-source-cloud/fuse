// Package debug provides debug nodes for workflow
package debug

import (
	"fmt"

	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// DebugProviderID is the ID of the debug node provider
const DebugProviderID = "fuse.io/workflows/internal/debug"

// NodeProvider is a debug node provider
type NodeProvider struct {
	workflow.NodeProvider

	nodes map[string]workflow.Node
}

// NewNodeProvider creates a new NodeProvider
func NewNodeProvider() workflow.NodeProvider {
	nodeProvider := &NodeProvider{
		nodes: make(map[string]workflow.Node),
	}

	nodeProvider.nodes[NilNodeID] = NewNilNode()

	return nodeProvider
}

// ID returns the ID of the NodeProvider
func (np *NodeProvider) ID() string {
	return DebugProviderID
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
