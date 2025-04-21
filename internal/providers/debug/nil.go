package debug

import (
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// NilNodeID is the ID of the nil node
const NilNodeID = "fuse.io/workflows/internal/debug/nil"

// NilNode is a nil node
type NilNode struct {
	workflow.Node
}

// NewNilNode creates a new nil node
func NewNilNode() workflow.Node {
	return &NilNode{}
}

// ID returns the ID of the nil node
func (n *NilNode) ID() string {
	return NilNodeID
}

// Metadata returns the metadata of the nil node
func (n *NilNode) Metadata() workflow.NodeMetadata {
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

// Execute executes the nil node
func (n *NilNode) Execute(_ workflow.NodeInput) (workflow.NodeResult, error) {
	return workflow.NewNodeResult(workflow.NodeOutputStatusSuccess, nil), nil
}
