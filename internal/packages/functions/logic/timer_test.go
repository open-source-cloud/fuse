package logic_test

import (
	"sync"
	"testing"

	"github.com/open-source-cloud/fuse/internal/packages/functions/logic"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// TestTimerFunction tests the TimerFunction function with a timer of 1000ms
func TestTimerFunction(t *testing.T) {
	timer := 1000
	input, err := workflow.NewFunctionInputWith(map[string]any{
		"timer": timer,
	})
	if err != nil {
		t.Fatalf("failed to create function input: %s", err)
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	execInfo := &workflow.ExecutionInfo{
		WorkflowID: "test-workflow",
		ExecID:     "test-exec",
		Input:      input,
		Finish: func(output workflow.FunctionOutput) {
			defer wg.Done()
			if output.Status != workflow.FunctionSuccess {
				t.Fatalf("timer function should return success, got %s", output.Status)
			}
		},
	}

	result, err := logic.TimerFunction(execInfo)
	if err != nil {
		t.Fatalf("failed to execute timer function: %v", err)
	}

	if !result.Async {
		t.Fatalf("timer function should be async")
	}

	if result.Output.Status != workflow.FunctionSuccess {
		t.Fatalf("timer function should return success, got %s", result.Output.Status)
	}

	wg.Wait()
}
