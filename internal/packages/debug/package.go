// Package debug provides debug nodes for workflow
package debug

import (
	"github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// PackageID is the ID of the debug function package
const PackageID = "fuse/internal/debug"

// New creates a new Package
func New() workflow.Package {
	return packages.NewInternal(PackageID, []workflow.Function{
		packages.NewInternalFunction(PackageID, NilFunctionID, NilFunctionMetadata(), NilFunction),
	})
}
