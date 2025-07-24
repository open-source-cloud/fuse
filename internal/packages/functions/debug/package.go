// Package debug provides debug nodes for workflow
package debug

import (
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// PackageID is the id of the debug function package
const PackageID = "fuse/pkg/debug"

// New creates a new debug Package
func New() *workflow.Package {
	return workflow.NewPackage(
		PackageID,
		workflow.NewFunction(NilFunctionID, NilFunctionMetadata(), NilFunction),
		workflow.NewFunction(PrintFunctionID, PrintFunctionMetadata(), PrintFunction),
	)
}
