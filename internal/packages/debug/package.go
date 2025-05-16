// Package debug provides debug nodes for workflow
package debug

import (
	"github.com/open-source-cloud/fuse/internal/packages"
)

// PackageID is the id of the debug function package
const PackageID = "fuse/pkg/debug"

// New creates a new Package
func New() packages.Package {
	return packages.NewInternal(PackageID, []packages.FunctionSpec{
		packages.NewInternalFunction(PackageID, NilFunctionID, NilFunctionMetadata(), NilFunction),
	})
}
