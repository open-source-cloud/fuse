package packages

import workflow "github.com/open-source-cloud/fuse/pkg/workflow"

// FunctionSpec represents an executable FunctionID and it's metadata
type FunctionSpec interface {
	ID() string
	Metadata() workflow.FunctionMetadata
	Execute(workflowID string, execID string, input *workflow.FunctionInput) (workflow.FunctionResult, error)
}
