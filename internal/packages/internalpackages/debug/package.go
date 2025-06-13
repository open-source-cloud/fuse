// Package debug provides debug nodes for workflow
package debug

import (
	"github.com/open-source-cloud/fuse/pkg/packages"
)

// PackageID is the id of the debug function package
const PackageID = "fuse/pkg/debug"

// New creates a new debug Package
func New() *packages.Package {
	return packages.NewPackage(
		PackageID,
		packages.NewFunction(NilFunctionID, NilFunctionMetadata(), NilFunction),
	)
}
