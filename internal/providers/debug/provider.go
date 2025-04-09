package debug

import "github.com/open-source-cloud/fuse/pkg/workflow"

const debugProviderID = "fuse.io/workflows/internal/debug"

type nodeProvider struct{}

func NewNodeProvider() workflow.NodeProvider {
	return &nodeProvider{}
}

func (p *nodeProvider) ID() string {
	return debugProviderID
}

func (p *nodeProvider) Nodes() []workflow.Node {
	return []workflow.Node{
		&nullNode{},
	}
}
