package logic_test

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/packages/functions/logic"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/suite"
)

type IfFunctionTestSuite struct {
	suite.Suite
}

func TestIfFunctionTestSuite(t *testing.T) {
	suite.Run(t, new(IfFunctionTestSuite))
}

// TestIfFunction tests the IfFunction function with a valid expression
func (s *IfFunctionTestSuite) TestIfFunction() {
	input, err := workflow.NewFunctionInputWith(map[string]any{
		"expression": "a > b",
		"a":          2,
		"b":          1,
	})
	if err != nil {
		s.Fail("failed to create function input: %s", err)
	}

	execInfo := &workflow.ExecutionInfo{
		WorkflowID: "test-workflow",
		Input:      input,
		ExecID:     "test-exec",
		Finish:     nil,
	}

	result, err := logic.IfFunction(execInfo)
	if err != nil {
		s.Fail("failed to execute if function: %v", err)
	}

	if result.Async {
		s.Fail("if function should not be async")
	}

	if result.Output.Status != workflow.FunctionSuccess {
		s.Fail("if function should return success, got %s", result.Output.Status)
	}
}
