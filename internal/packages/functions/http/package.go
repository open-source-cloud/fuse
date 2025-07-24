// Package http provides http functions for workflow
package http

import "github.com/open-source-cloud/fuse/pkg/workflow"

// PackageID is the id of the http package
const PackageID = "fuse/pkg/http"

// New creates a new http package
func New() *workflow.Package {
	return workflow.NewPackage(
		PackageID,
		workflow.NewFunction(
			HTTPFunctionID,
			RequestFunctionMetadata(),
			RequestFunction,
		),
	)
}
