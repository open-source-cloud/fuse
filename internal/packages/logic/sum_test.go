package logic_test

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/packages/logic"
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

	sumFunction, err := logic.SumFunction(nil, input)
	if err != nil {
		t.Fatalf("sum function should return success, got %s", err)
	}

	if sumFunction.Async {
		t.Fatalf("sum function should not be async")
	}

	if sumFunction.Output.Status != workflow.FunctionSuccess {
		t.Fatalf("sum function should return success, got %s", sumFunction.Output.Status)
	}

	if sumFunction.Output.Data["sum"] != expected {
		t.Fatalf("sum function should return %f, got %f", expected, sumFunction.Output.Data["sum"])
	}
}
