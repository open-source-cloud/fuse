package logic

import (
	"crypto/rand"
	"fmt"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"math/big"
)

type randNode struct{}

func (n *randNode) ID() string {
	return fmt.Sprintf("%s/rand", logicProviderID)
}

func (n *randNode) Metadata() workflow.NodeMetadata {
	return workflow.NewNodeMetadata(
		// input
		workflow.InputOutputMetadata{},
		// output
		workflow.InputOutputMetadata{
			Parameters: workflow.Parameters{
				"rand": workflow.ParameterSchema{
					Name:        "rand",
					Type:        "int",
					Validations: nil,
					Description: "Generated random number",
				},
			},
			Edges: workflow.EdgeMetadata{},
		},
	)
}

func (n *randNode) Execute(_ workflow.NodeInput) (workflow.NodeResult, error) {
	randomNumberBig, err := rand.Int(rand.Reader, big.NewInt(1000))
	if err != nil {
		return nil, fmt.Errorf("failed to generate random number: %w", err)
	}

	randomNumber := int(randomNumberBig.Int64())

	return workflow.NewNodeResult(workflow.NodeOutputStatusSuccess, map[string]any{
		"rand": randomNumber,
	}), nil
}
