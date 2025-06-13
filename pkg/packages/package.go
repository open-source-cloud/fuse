// Package packages workflow function packages
package packages

import "github.com/open-source-cloud/fuse/pkg/workflow"

// NewPackage creates a new Package
func NewPackage(id string, functions ...*Function) *Package {
	return &Package{
		ID:        id,
		Functions: functions,
	}
}

// NewFunction creates a new packaged Function
func NewFunction(id string, metadata workflow.FunctionMetadata, fn workflow.Function) *Function {
	return &Function{
		ID:       id,
		Metadata: metadata,
		Function: fn,
	}
}

type (
	// Package workflow function Package
	Package struct {
		ID        string     `json:"id"`
		Functions []*Function `json:"functions"`
	}

	// Function packaged Function
	Function struct {
		ID       string                    `json:"id"`
		Metadata workflow.FunctionMetadata `json:"metadata"`
		Function workflow.Function
	}
)
