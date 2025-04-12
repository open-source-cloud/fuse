package logic

import (
	"fmt"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/rs/zerolog/log"
	"math/rand"
)

type randNode struct{}

func (n *randNode) ID() string {
	return fmt.Sprintf("%s/sum", logicProviderID)
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
	log.Info().Msgf("RandNode executed with input: %s", input)

	randomNumber := rand.Intn(100)
	return workflow.NewNodeResult(workflow.NodeOutputStatusSuccess, map[string]interface{}{
		"rand": randomNumber,
	}), nil
}
