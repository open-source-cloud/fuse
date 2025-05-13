package packages

import (
	"fmt"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

type functionCallable func(input *workflow.FunctionInput) (workflow.FunctionResult, error)

type internalFunction struct {
	workflow.Function

	id       string
	metadata workflow.FunctionMetadata
	fn       functionCallable
}

// NewInternalFunction creates a new internal function
func NewInternalFunction(packageID string, id string, metadata workflow.FunctionMetadata, fn functionCallable) workflow.Function {
	return &internalFunction{
		id:       fmt.Sprintf("%s/%s", packageID, id),
		metadata: metadata,
		fn:       fn,
	}
}

func (f *internalFunction) ID() string {
	return f.id
}

func (f *internalFunction) Metadata() workflow.FunctionMetadata {
	return f.metadata
}

func (f *internalFunction) Execute(input map[string]any) (workflow.FunctionResult, error) {
	fnInput, err := workflow.InitFunctionInput(input)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize input store: %w", err)
	}
	return f.fn(fnInput)
}
