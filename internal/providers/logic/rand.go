package logic

import (
	"fmt"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"math/rand"
)

type randNode struct{}

func (n *randNode) ID() string {
	return fmt.Sprintf("%s/rand", logicProviderID)
}

func (n *randNode) Metadata() workflow.NodeMetadata {
	return workflow.NewNodeMetadata(
		// input
		workflow.InputOutputMetadata{
			Parameters: workflow.Parameters{},
		},
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

func (n *randNode) Execute(input workflow.NodeInput) (workflow.NodeResult, error) {
	randomNumber := rand.Intn(100)
	return workflow.NewNodeResult(workflow.NodeOutputStatusSuccess, map[string]any{
		"rand": randomNumber,
	}), nil
}
