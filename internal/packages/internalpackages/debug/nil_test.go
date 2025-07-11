package debug_test

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/packages/debug"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// TestNilFunction tests the NilFunction function
func TestNilFunction(t *testing.T) {
	execInfo := &workflow.ExecutionInfo{}
	input, err := workflow.NewFunctionInputWith(map[string]any{})
	if err != nil {
		t.Fatalf("failed to create function input: %s", err)
	}

	result, err := debug.NilFunction(execInfo, input)
	if err != nil {
		t.Fatalf("failed to execute nil function: %v", err)
	}

	if result.Async {
		t.Fatalf("nil function should not be async")
	}

	if result.Output.Status != workflow.FunctionSuccess {
		t.Fatalf("nil function should return success, got %s", result.Output.Status)
	}
}
