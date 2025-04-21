package debug

import (
	"fmt"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

type nilNode struct{}

func (n *nilNode) ID() string {
	return fmt.Sprintf("%s/nil", debugProviderID)
}

func (n *nilNode) Metadata() workflow.NodeMetadata {
	return workflow.NewNodeMetadata(
		workflow.InputOutputMetadata{
			Parameters: workflow.Parameters{},
			Edges:      workflow.EdgeMetadata{},
		},
		workflow.InputOutputMetadata{
			Parameters: workflow.Parameters{},
			Edges:      workflow.EdgeMetadata{},
		},
	)
}

func (n *nilNode) Execute(_ workflow.NodeInput) (workflow.NodeResult, error) {
	return workflow.NewNodeResult(workflow.NodeOutputStatusSuccess, nil), nil
}
