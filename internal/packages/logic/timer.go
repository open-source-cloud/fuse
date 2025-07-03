package logic

import (
	"context"
	"time"

	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// TimerFunctionID is the id of the timer function
const TimerFunctionID = "timer"

// TimerFunctionMetadata returns the metadata of the timer function
func TimerFunctionMetadata() workflow.FunctionMetadata {
	return workflow.FunctionMetadata{
		Input: workflow.InputMetadata{
			Parameters: workflow.Parameters{
				"timer": workflow.ParameterSchema{
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
func TimerFunction(execInfo *workflow.ExecutionInfo, input *workflow.FunctionInput) (workflow.FunctionResult, error) {
	ctx, cancel := context.WithCancel(context.Background())

	timer := input.GetInt("timer")
	duration := time.Duration(timer) * time.Millisecond

	go func() {
		ticker := time.NewTicker(duration)
		for {
			select {
			case <-ticker.C:
				execInfo.Finish(workflow.NewFunctionSuccessOutput(nil))
				cancel()
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()

	return workflow.NewFunctionResultAsync(), nil
}
