package logic

import (
	"context"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"time"
)

// TimerNodeID is the ID of the timer node
const TimerNodeID = "fuse.io/workflows/internal/logic/timer"

// TimerNode is a timer node
type TimerNode struct {
	workflow.Node
}

// NewTimerNode creates a new timer node
func NewTimerNode() workflow.Node {
	return &TimerNode{}
}

// ID returns the ID of the timer node
func (n *TimerNode) ID() string {
	return TimerNodeID
}

// Metadata returns the metadata of the timer node
func (n *TimerNode) Metadata() workflow.NodeMetadata {
	return workflow.NewNodeMetadata(
		workflow.InputMetadata{
			Parameters: workflow.Parameters{
				"values": workflow.ParameterSchema{
					Name:        "timer",
					Type:        "int",
					Required:    true,
					Validations: nil,
					Description: "Timer in ms",
					Default:     0,
				},
			},
		},
		workflow.OutputMetadata{},
	)
}

// Execute executes timer node
func (n *TimerNode) Execute(input *workflow.NodeInput) (workflow.NodeResult, error) {
	ctx, cancel := context.WithCancel(context.Background())
	duration := time.Duration(input.Get("timer").(int)) * time.Millisecond
	resultChan := make(chan workflow.NodeOutput)

	go func() {
		ticker := time.NewTicker(duration)
		for {
			select {
			case <-ticker.C:
				resultChan <- workflow.NewNodeOutput(workflow.NodeOutputStatusSuccess, nil)
				cancel()
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()

	return workflow.NewNodeResultAsync(resultChan), nil
}
