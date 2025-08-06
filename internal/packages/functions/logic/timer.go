package logic

import (
	"context"
	"time"

	"github.com/open-source-cloud/fuse/internal/packages/transport"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/rs/zerolog/log"
)

// TimerFunctionID is the id of the timer function
const TimerFunctionID = "timer"

// TimerFunctionMetadata returns the metadata of the timer function
func TimerFunctionMetadata() workflow.FunctionMetadata {
	return workflow.FunctionMetadata{
		Transport: transport.Internal,
		Input: workflow.InputMetadata{
			CustomParameters: false,
			Parameters: []workflow.ParameterSchema{
				{
					Name:        "timer",
					Type:        "int",
					Required:    true,
					Validations: nil,
					Description: "Timer in ms",
					Default:     0,
				},
			},
			Edges: workflow.InputEdgeMetadata{
				Count:      0,
				Parameters: make([]workflow.ParameterSchema, 0),
			},
		},
		Output: workflow.OutputMetadata{
			Parameters: make([]workflow.ParameterSchema, 0),
			Edges:      make([]workflow.OutputEdgeMetadata, 0),
		},
	}
}

// TimerFunction executes timer function
func TimerFunction(execInfo *workflow.ExecutionInfo) (workflow.FunctionResult, error) {
	timer := execInfo.Input.GetInt("timer")

	if timer == 0 {
		log.Error().Msg("timer is 0, skipping timer function")
		return workflow.NewFunctionResultAsync(), nil
	}

	duration := time.Duration(timer) * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())

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
