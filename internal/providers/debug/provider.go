// Package debug provides debug nodes for workflow
package debug

import "github.com/open-source-cloud/fuse/pkg/workflow"

const debugProviderID = "fuse.io/workflows/internal/debug"

type nodeProvider struct{}

// NewNodeProvider creates a new DebugNodeProvider
func NewNodeProvider() workflow.NodeProvider {
	return &nodeProvider{}
}

func (p *nodeProvider) ID() string {
	return debugProviderID
}

func (p *nodeProvider) Nodes() []workflow.Node {
	return []workflow.Node{
		&nilNode{},
	}
}
