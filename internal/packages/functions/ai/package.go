// Package ai provides AI agent functions for workflow
package ai

import (
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// PackageID is the id of the AI agent function package
const PackageID = "fuse/pkg/ai"

// New creates a new AI agent Package
func New() *workflow.Package {
	return workflow.NewPackage(
		PackageID,
		workflow.NewFunction(StreamLLMFunctionID, StreamLLMFunctionMetadata(), StreamLLMFunction),
	)
}
