package workflow_test

import (
	"testing"

	"github.com/open-source-cloud/fuse/pkg/workflow"
)

func TestFunctionInput(t *testing.T) {
	input, err := workflow.NewFunctionInputWith(map[string]any{
		"a": 1,
	})

	if err != nil {
		t.Fatalf("failed to create function input: %s", err)
	}

	if input.Get("a") != 1 {
		t.Fatalf("function input should return 1, got %d", input.Get("a"))
	}

	if input.GetStr("b") != "" {
		t.Fatalf("function input should return empty string, got %s", input.GetStr("b"))
	}

	if input.GetInt("c") != 0 {
		t.Fatalf("function input should return 0, got %d", input.GetInt("c"))
	}
}

func TestFunctionInput_GetAnySliceOrDefault(t *testing.T) {
	input, err := workflow.NewFunctionInputWith(map[string]any{
		"a": []any{1, 2, 3},
	})

	if err != nil {
		t.Fatalf("failed to create function input: %s", err)
	}

	slice := input.GetAnySliceOrDefault("a", []any{4, 5, 6})

	if len(slice) != 3 {
		t.Fatalf("function input should return 3, got %d", len(slice))
	}

	if slice[0] != 1 {
		t.Fatalf("function input should return 1, got %d", slice[0])
	}
}
