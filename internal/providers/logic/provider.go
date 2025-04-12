package logic

import "github.com/open-source-cloud/fuse/pkg/workflow"

const logicProviderID = "fuse.io/workflows/internal/logic"

type nodeProvider struct{}

func NewNodeProvider() workflow.NodeProvider {
	return &nodeProvider{}
}

func (p *nodeProvider) ID() string {
	return logicProviderID
}

func (p *nodeProvider) Nodes() []workflow.Node {
	return []workflow.Node{
		&sumNode{},
		&randNode{},
	}
}
