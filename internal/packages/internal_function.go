package packages

import (
	"fmt"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

type internalFunction struct {
	id       string
	metadata workflow.FunctionMetadata
	fn       workflow.Function
}

// NewInternalFunction creates a new internal function
func NewInternalFunction(packageID string, id string, metadata workflow.FunctionMetadata, fn workflow.Function) FunctionSpec {
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

func (f *internalFunction) Execute(input *workflow.FunctionInput) (workflow.FunctionResult, error) {
	return f.fn(input)
}
