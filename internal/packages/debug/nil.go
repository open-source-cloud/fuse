package debug

import (
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

const NilFunctionID = "nil"

func NilFunctionMetadata() workflow.FunctionMetadata {
	return workflow.NewFunctionMetadata(
		workflow.InputMetadata{},
		workflow.OutputMetadata{},
	)
}

// NilFunction executes the nil function
func NilFunction(_ *workflow.FunctionInput) (workflow.FunctionResult, error) {
	return workflow.NewFunctionResult(workflow.FunctionSuccess, nil), nil
}
