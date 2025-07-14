package logic_test

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/packages/functions/logic"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// TestIfFunction tests the IfFunction function with a valid expression
func TestIfFunction(t *testing.T) {

	input, err := workflow.NewFunctionInputWith(map[string]any{
		"expression": "a > b",
		"a":          2,
		"b":          1,
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

	result, err := logic.IfFunction(execInfo)
	if err != nil {
		t.Fatalf("failed to execute if function: %v", err)
	}

	if result.Async {
		t.Fatalf("if function should not be async")
	}

	if result.Output.Status != workflow.FunctionSuccess {
		t.Fatalf("if function should return success, got %s", result.Output.Status)
	}
}
