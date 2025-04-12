package logic

import (
	"fmt"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/rs/zerolog/log"
)

type sumNode struct{}

func (n *sumNode) ID() string {
	return fmt.Sprintf("%s/sum", logicProviderID)
}

func (n *sumNode) Metadata() workflow.NodeMetadata {
	return workflow.NewNodeMetadata(
		// input
		workflow.InputOutputMetadata{
			Parameters: workflow.Parameters{
				"value": workflow.ParameterSchema{
					Name:        "value",
					Type:        "[]int",
					Required:    true,
					Validations: nil,
					Description: "Value to sum",
					Default:     0,
				},
			},
			Edges: workflow.EdgeMetadata{
				Count:      workflow.EdgesUnlimited,
				Parameters: workflow.Parameters{},
			},
		},
		// output
		workflow.InputOutputMetadata{
			Parameters: workflow.Parameters{
				"result": workflow.ParameterSchema{
					Name:        "result",
					Type:        "int",
					Validations: nil,
					Description: "Result of the sum",
					Default:     0,
				},
			},
			Edges: workflow.EdgeMetadata{
				Count:      workflow.EdgesUnlimited,
				Parameters: workflow.Parameters{},
			},
		},
	)
}

func (n *sumNode) Execute(input workflow.NodeInput) (workflow.NodeResult, error) {
	log.Info().Msgf("SumNode executed with input: %s", input)
	return workflow.NewNodeResult(workflow.NodeOutputStatusSuccess, nil), nil
}
