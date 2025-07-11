package logic_test

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/packages/logic"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// TestTimerFunction tests the TimerFunction function with a timer of 1000ms
func TestTimerFunction(t *testing.T) {
	execInfo := &workflow.ExecutionInfo{
		WorkflowID: "test-workflow",
		ExecID:     "test-exec",
		Finish: func(output workflow.FunctionOutput) {
			t.Logf("timer function finished: %v", output)
		},
	}

	timer := 1000
	input, err := workflow.NewFunctionInputWith(map[string]any{
		"timer": timer,
	})
	if err != nil {
		t.Fatalf("failed to create function input: %s", err)
	}

	result, err := logic.TimerFunction(execInfo, input)
	if err != nil {
		t.Fatalf("failed to execute timer function: %v", err)
	}

	if !result.Async {
		t.Fatalf("timer function should be async")
	}

	if result.Output.Status != workflow.FunctionSuccess {
		t.Fatalf("timer function should return success, got %s", result.Output.Status)
	}
}
