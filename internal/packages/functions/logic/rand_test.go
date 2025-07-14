package logic_test

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/packages/functions/logic"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// TestRandFunction tests the RandFunction function with min and max parameters
func TestRandFunction(t *testing.T) {
	input, err := workflow.NewFunctionInputWith(map[string]any{
		"min": 1,
		"max": 100,
	})
	if err != nil {
		t.Fatalf("failed to create function input: %s", err)
	}

	execInfo := &workflow.ExecutionInfo{
		WorkflowID: "test-workflow",
		Input:      input,
		ExecID:     "test-exec",
		Finish:     nil,
	}

	result, err := logic.RandFunction(execInfo)
	if err != nil {
		t.Fatalf("failed to execute rand function: %v", err)
	}

	if result.Async {
		t.Fatalf("rand function should not be async")
	}

	if result.Output.Status != workflow.FunctionSuccess {
		t.Fatalf("rand function should return success, got %s", result.Output.Status)
	}

	rand := result.Output.Data["rand"].(int)
	if rand < 1 || rand > 100 {
		t.Fatalf("rand function should return a number between 1 and 100, got %d", rand)
	}
}
