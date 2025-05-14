package logic

import (
	"context"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"time"
)

// TimerFunctionID is the ID of the timer function
const TimerFunctionID = "timer"

// TimerFunctionMetadata returns the metadata of the timer function
func TimerFunctionMetadata() workflow.FunctionMetadata {
	return workflow.FunctionMetadata{
		Input: workflow.InputMetadata{
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
	}
}

// TimerFunction executes timer function
func TimerFunction(input *workflow.FunctionInput) (workflow.FunctionResult, error) {
	ctx, cancel := context.WithCancel(context.Background())
	duration := time.Duration(input.Get("timer").(int)) * time.Millisecond
	resultChan := make(chan workflow.FunctionOutput)

	go func() {
		ticker := time.NewTicker(duration)
		for {
			select {
			case <-ticker.C:
				resultChan <- workflow.NewFunctionOutput(workflow.FunctionSuccess, nil)
				cancel()
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()

	return workflow.NewFunctionResultAsync(), nil
}
