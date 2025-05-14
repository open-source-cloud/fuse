package debug

import (
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

const NilFunctionID = "nil"

func NilFunctionMetadata() workflow.FunctionMetadata {
	return workflow.FunctionMetadata{}
}

// NilFunction executes the nil function
func NilFunction(_ *workflow.FunctionInput) (workflow.FunctionResult, error) {
	return workflow.NewFunctionResultSuccess(), nil
}
