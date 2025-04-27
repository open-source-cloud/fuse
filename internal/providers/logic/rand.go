package logic

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// RandNodeID is the ID of the rand node
const RandNodeID = "fuse.io/workflows/internal/logic/rand"

// RandNode is a rand node
type RandNode struct {
	workflow.Node
}

// NewRandNode creates a new rand node
func NewRandNode() workflow.Node {
	return &RandNode{}
}

// ID returns the ID of the rand node
func (n *RandNode) ID() string {
	return RandNodeID
}

// Metadata returns the metadata of the rand node
func (n *RandNode) Metadata() workflow.NodeMetadata {
	return workflow.NewNodeMetadata(
		// input
		workflow.InputMetadata{},
		// output
		workflow.OutputMetadata{
			Parameters: workflow.Parameters{
				"rand": workflow.ParameterSchema{
					Name:        "rand",
					Type:        "int",
					Validations: nil,
					Description: "Generated random number",
				},
			},
		},
	)
}

// Execute executes the rand node and returns a random number
func (n *RandNode) Execute(_ *workflow.NodeInput) (workflow.NodeResult, error) {
	randomNumberBig, err := rand.Int(rand.Reader, big.NewInt(1000))
	if err != nil {
		return nil, fmt.Errorf("failed to generate random number: %w", err)
	}

	randomNumber := int(randomNumberBig.Int64())

	return workflow.NewNodeResult(workflow.NodeOutputStatusSuccess, map[string]any{
		"rand": randomNumber,
	}), nil
}
