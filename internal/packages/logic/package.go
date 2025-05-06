// Package logic provides logic nodes for workflows
package logic

import (
	"github.com/open-source-cloud/fuse/internal/packages"

	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// PackageID is the ID of the debug function package
const PackageID = "fuse/internal/logic"

// New creates a new Package
func New() workflow.Package {
	return packages.NewInternal(PackageID, []workflow.Function{
		packages.NewInternalFunction(PackageID, IfFunctionID, IfFunctionMetadata(), IfFunction),
		packages.NewInternalFunction(PackageID, RandFunctionID, RandFunctionMetadata(), RandFunction),
		packages.NewInternalFunction(PackageID, SumFunctionID, SumFunctionMetadata(), SumFunction),
		packages.NewInternalFunction(PackageID, TimerFunctionID, TimerFunctionMetadata(), TimerFunction),
	})
}
