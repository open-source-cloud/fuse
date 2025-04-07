package debug

import "github.com/open-source-cloud/fuse/pkg/workflow"

const debugProviderID = "fuse.io/workflows/internal/debug"

type NodeProvider struct{}

func NewNodeProvider() *NodeProvider {
	return &NodeProvider{}
}

func (p *NodeProvider) ID() string {
	return debugProviderID
}

func (p *NodeProvider) Nodes() []workflow.Node {
	return []workflow.Node{
		&NullNode{},
		&LogNode{},
	}
}
