package logic

import (
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// SumNodeID is the ID of the sum node
const SumNodeID = "fuse.io/workflows/internal/logic/sum"

// SumNode is a sum node
type SumNode struct {
	workflow.Node
}

// NewSumNode creates a new sum node
func NewSumNode() workflow.Node {
	return &SumNode{}
}

// ID returns the ID of the sum node
func (n *SumNode) ID() string {
	return SumNodeID
}

// Metadata returns the metadata of the sum node
func (n *SumNode) Metadata() workflow.NodeMetadata {
	return workflow.NewNodeMetadata(
		// input
		workflow.InputMetadata{
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
			Edges: workflow.InputEdgeMetadata{
				Parameters: workflow.Parameters{},
			},
		},
		// output
		workflow.OutputMetadata{
			Parameters: workflow.Parameters{
				"result": workflow.ParameterSchema{
					Name:        "sum",
					Type:        "int",
					Validations: nil,
					Description: "Result of the sum",
					Default:     0,
				},
			},
		},
	)
}

// Execute executes the sum node and returns the sum of the values
func (n *SumNode) Execute(input *workflow.NodeInput) (workflow.NodeResult, error) {
	sum := 0
	values := input.Get("values").([]any)
	for _, value := range values {
		intValue, _ := value.(int)
		sum += intValue
	}

	return workflow.NewNodeResult(workflow.NodeOutputStatusSuccess, map[string]any{"sum": sum}), nil
}
