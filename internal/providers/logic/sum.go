package logic

import (
	"fmt"
	"github.com/open-source-cloud/fuse/pkg/workflow"
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
				"values": workflow.ParameterSchema{
					Name:        "values",
					Type:        "[]int",
					Required:    true,
					Validations: nil,
					Description: "Values to sum",
					Default:     []int{},
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
					Name:        "sum",
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
	sum := 0
	values := input["values"].([]any)
	for _, value := range values {
		intValue, _ := value.(int)
		sum += intValue
	}

	return workflow.NewNodeResult(workflow.NodeOutputStatusSuccess, map[string]any{"sum": sum}), nil
}
