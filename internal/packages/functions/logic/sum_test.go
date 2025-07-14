package logic_test

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/packages/functions/logic"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// TestSumFunction tests the SumFunction
func TestSumFunction(t *testing.T) {
	expected := 15.0
	values := []float64{1, 2, 3, 4, 5}

	input, err := workflow.NewFunctionInputWith(map[string]any{
		"values": values,
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

	result, err := logic.SumFunction(execInfo)
	if err != nil {
		t.Fatalf("sum function should return success, got %s", err)
	}

	if result.Async {
		t.Fatalf("sum function should not be async")
	}

	if result.Output.Status != workflow.FunctionSuccess {
		t.Fatalf("sum function should return success, got %s", result.Output.Status)
	}

	if result.Output.Data["sum"] != expected {
		t.Fatalf("sum function should return %f, got %f", expected, result.Output.Data["sum"])
	}
}
